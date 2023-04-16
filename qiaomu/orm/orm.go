package orm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	qlog "github.com/qingbo1011/qiaomu/log"
	"github.com/qingbo1011/qiaomu/utils"
)

type QueenDB struct {
	db     *sql.DB
	logger *qlog.Logger
	Prefix string
}

type QueenSession struct {
	db          *QueenDB
	tx          *sql.Tx
	beginTx     bool
	tableName   string
	fieldName   []string
	placeHolder []string
	values      []any
	updateParam strings.Builder
	whereParam  strings.Builder
	whereValues []any
}

// Open 打开DB链接
func Open(driverName string, source string) *QueenDB {
	db, err := sql.Open(driverName, source)
	if err != nil {
		panic(err)
	}
	db.SetMaxIdleConns(5)                  // 设置最大空闲连接数(默认为2)
	db.SetMaxOpenConns(100)                // 最设置大连接数(默认不限制)
	db.SetConnMaxLifetime(time.Minute * 3) // 连接最大存活时间
	db.SetConnMaxIdleTime(time.Minute * 1) //空闲连接最大存活时间
	queenDb := &QueenDB{
		db:     db,
		logger: qlog.Default(),
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	return queenDb
}

// Close 关闭DB链接
func (db *QueenDB) Close() error {
	return db.db.Close()
}

// SetMaxIdleConns 设置最大空闲连接数
func (db *QueenDB) SetMaxIdleConns(n int) {
	db.db.SetMaxIdleConns(n)
}

// New 创建session(用户通过ORM进行每一个操作都应该是在session中独立完成的)
func (db *QueenDB) New(data any) *QueenSession {
	session := &QueenSession{
		db: db,
	}
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer {
		panic(errors.New("data must be pointer"))
	}
	tVar := t.Elem()
	if session.tableName == "" {
		session.tableName = session.db.Prefix + strings.ToLower(Name(tVar.Name()))
	}
	return session
}

// Name 名称处理(eg:UserName处理为 User_Name。代码中往往是驼峰格式，而数据库中需要为下划线格式。转小写操作在fieldNames中)
func Name(name string) string {
	var result strings.Builder
	for i, char := range name {
		if unicode.IsUpper(char) && i != 0 {
			result.WriteRune('_')
		}
		result.WriteRune(char)
	}
	return result.String()
}

// Table 设置数据库表的名称
func (s *QueenSession) Table(name string) *QueenSession {
	s.tableName = name
	return s
}

// Insert 插入数据，返回主键id、影响条数和error
func (s *QueenSession) Insert(data any) (int64, int64, error) {
	s.fieldNames(data) // 处理字段名
	// insert into table (xxx,xxx) values(?,?)
	query := fmt.Sprintf("insert into %s (%s) values (%s)", s.tableName, strings.Join(s.fieldName, ","), strings.Join(s.placeHolder, ","))
	s.db.logger.Info(query)
	var stmt *sql.Stmt
	var err error
	if s.beginTx {
		stmt, err = s.tx.Prepare(query)
	} else {
		stmt, err = s.db.db.Prepare(query)
	}
	if err != nil {
		s.db.logger.Error(err)
		return -1, -1, err
	}
	r, err := stmt.Exec(s.values...)
	if err != nil {
		s.db.logger.Error(err)
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		s.db.logger.Error(err)
		return -1, -1, err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		s.db.logger.Error(err)
		return -1, -1, err
	}
	s.db.logger.Info(utils.ConcatenatedString([]string{"新增记录主键id为：", strconv.Itoa(int(id)), ";\t影响条数：", strconv.Itoa(int(affected))}))
	return id, affected, nil
}

// 通过反射和tag检测，根据结构体的成员变量名判断出插入字段名
func (s *QueenSession) fieldNames(data any) {
	// 反射
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)
	if t.Kind() != reflect.Pointer {
		panic(errors.New("data must be pointer"))
	}
	tVar := t.Elem()
	vVar := v.Elem()
	if s.tableName == "" {
		s.tableName = s.db.Prefix + strings.ToLower(Name(tVar.Name()))
	}
	for i := 0; i < tVar.NumField(); i++ {
		fieldName := tVar.Field(i).Name
		tag := tVar.Field(i).Tag
		sqlTag := tag.Get("qorm") // 获取 tag
		if sqlTag == "" {
			sqlTag = strings.ToLower(Name(fieldName))
		} else {
			if strings.Contains(sqlTag, "auto_increment") { // 自增主键
				continue
			}
			if strings.Contains(sqlTag, ",") {
				sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
			}
		}
		id := vVar.Field(i).Interface()
		if strings.ToLower(sqlTag) == "id" && IsAutoId(id) {
			continue
		}
		s.fieldName = append(s.fieldName, sqlTag)
		s.placeHolder = append(s.placeHolder, "?")
		s.values = append(s.values, vVar.Field(i).Interface())
	}
}

// IsAutoId 判断id是否为自增id
func IsAutoId(id any) bool {
	t := reflect.TypeOf(id)
	switch t.Kind() {
	case reflect.Int64:
		if id.(int64) <= 0 {
			return true
		}
	case reflect.Int32:
		if id.(int32) <= 0 {
			return true
		}
	case reflect.Int:
		if id.(int) <= 0 {
			return true
		}
	default:
		return false
	}
	return false
}

// InsertBatch 批量插入
func (s *QueenSession) InsertBatch(data []any) (int64, int64, error) {
	// insert into table (xxx,xxx) values(?,?),(?,?)
	if len(data) == 0 {
		s.db.logger.Error(errors.New("no data insert"))
		return -1, -1, errors.New("no data insert")
	}
	s.fieldNames(data[0])
	query := fmt.Sprintf("insert into %s (%s) values ", s.tableName, strings.Join(s.fieldName, ","))
	var sb strings.Builder
	sb.WriteString(query)
	for index, _ := range data {
		sb.WriteString("(")
		sb.WriteString(strings.Join(s.placeHolder, ","))
		sb.WriteString(")")
		if index < len(data)-1 {
			sb.WriteString(",")
		}
	}
	s.batchValues(data)
	s.db.logger.Info(sb.String())
	var stmt *sql.Stmt
	var err error
	if s.beginTx {
		stmt, err = s.tx.Prepare(sb.String())
	} else {
		stmt, err = s.db.db.Prepare(sb.String())
	}

	if err != nil {
		s.db.logger.Error(err)
		return -1, -1, err
	}
	r, err := stmt.Exec(s.values...)
	if err != nil {
		s.db.logger.Error(err)
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		s.db.logger.Error(err)
		return -1, -1, err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		s.db.logger.Error(err)
		return -1, -1, err
	}
	s.db.logger.Info(utils.ConcatenatedString([]string{"批量插入数据的第一题条记录主键id为：", strconv.Itoa(int(id)), ";\t影响条数：", strconv.Itoa(int(affected))}))
	return id, affected, nil
}

// 处理批量插入数据的value
func (s *QueenSession) batchValues(data []any) {
	s.values = make([]any, 0)
	for _, v := range data {
		t := reflect.TypeOf(v)
		v := reflect.ValueOf(v)
		if t.Kind() != reflect.Pointer {
			panic(errors.New("data must be pointer"))
		}
		tVar := t.Elem()
		vVar := v.Elem()
		for i := 0; i < tVar.NumField(); i++ {
			fieldName := tVar.Field(i).Name
			tag := tVar.Field(i).Tag
			sqlTag := tag.Get("qorm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(Name(fieldName))
			} else {
				if strings.Contains(sqlTag, "auto_increment") { // 自增长的id主键
					continue
				}
			}
			id := vVar.Field(i).Interface()
			if strings.ToLower(sqlTag) == "id" && IsAutoId(id) {
				continue
			}
			s.values = append(s.values, vVar.Field(i).Interface())
		}
	}
}

// Update 更新数据
func (s *QueenSession) Update(data ...any) (int64, int64, error) {
	// Update("age",1) or Update(user)
	if len(data) > 2 {
		return -1, -1, errors.New("param not valid")
	}
	if len(data) == 0 {
		query := fmt.Sprintf("update %s set %s", s.tableName, s.updateParam.String())
		var sb strings.Builder
		sb.WriteString(query)
		sb.WriteString(s.whereParam.String())
		s.db.logger.Info(sb.String())
		var stmt *sql.Stmt
		var err error
		if s.beginTx {
			stmt, err = s.tx.Prepare(sb.String())
		} else {
			stmt, err = s.db.db.Prepare(sb.String())
		}
		if err != nil {
			return -1, -1, err
		}
		s.values = append(s.values, s.whereValues...)
		r, err := stmt.Exec(s.values...)
		if err != nil {
			return -1, -1, err
		}
		id, err := r.LastInsertId()
		if err != nil {
			return -1, -1, err
		}
		affected, err := r.RowsAffected()
		if err != nil {
			return -1, -1, err
		}
		return id, affected, nil
	}
	single := true
	if len(data) == 2 {
		single = false
	}
	// update table set age=?,name=? where id=?
	if !single {
		if s.updateParam.String() != "" {
			s.updateParam.WriteString(",")
		}
		s.updateParam.WriteString(data[0].(string))
		s.updateParam.WriteString(" = ? ")
		s.values = append(s.values, data[1])
	} else {
		updateData := data[0]
		t := reflect.TypeOf(updateData)
		v := reflect.ValueOf(updateData)
		if t.Kind() != reflect.Pointer {
			panic(errors.New("updateData must be pointer"))
		}
		tVar := t.Elem()
		vVar := v.Elem()
		for i := 0; i < tVar.NumField(); i++ {
			fieldName := tVar.Field(i).Name
			tag := tVar.Field(i).Tag
			sqlTag := tag.Get("msorm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(Name(fieldName))
			} else {
				if strings.Contains(sqlTag, "auto_increment") {
					//自增长的主键id
					continue
				}
				if strings.Contains(sqlTag, ",") {
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
				}
			}
			id := vVar.Field(i).Interface()
			if strings.ToLower(sqlTag) == "id" && IsAutoId(id) {
				continue
			}
			if s.updateParam.String() != "" {
				s.updateParam.WriteString(",")
			}
			s.updateParam.WriteString(sqlTag)
			s.updateParam.WriteString(" = ? ")
			s.values = append(s.values, vVar.Field(i).Interface())
		}
	}
	query := fmt.Sprintf("update %s set %s", s.tableName, s.updateParam.String())
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())
	var stmt *sql.Stmt
	var err error
	if s.beginTx {
		stmt, err = s.tx.Prepare(sb.String())
	} else {
		stmt, err = s.db.db.Prepare(sb.String())
	}
	if err != nil {
		return -1, -1, err
	}
	s.values = append(s.values, s.whereValues...)
	r, err := stmt.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	return id, affected, nil
}
