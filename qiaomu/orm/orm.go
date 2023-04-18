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
		s.db.logger.Error(errors.New("param not valid"))
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
			s.db.logger.Error(err)
			return -1, -1, err
		}
		s.values = append(s.values, s.whereValues...)
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
		s.db.logger.Info(utils.ConcatenatedString([]string{"影响条数：", strconv.Itoa(int(affected))}))
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
			sqlTag := tag.Get("qorm")
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
		s.db.logger.Error(err)
		return -1, -1, err
	}
	s.values = append(s.values, s.whereValues...)
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
	s.db.logger.Info(utils.ConcatenatedString([]string{"影响条数：", strconv.Itoa(int(affected))}))
	return id, affected, nil
}

// UpdateParam 指定字段名和value进行更新
func (s *QueenSession) UpdateParam(field string, value any) *QueenSession {
	if s.updateParam.String() != "" {
		s.updateParam.WriteString(",")
	}
	s.updateParam.WriteString(field)
	s.updateParam.WriteString(" = ? ")
	s.values = append(s.values, value)
	return s
}

// UpdateMap 根据map更新数据(key为字段名，value为更新数据)
func (s *QueenSession) UpdateMap(data map[string]any) *QueenSession {
	for k, v := range data {
		if s.updateParam.String() != "" {
			s.updateParam.WriteString(",")
		}
		s.updateParam.WriteString(k)
		s.updateParam.WriteString(" = ? ")
		s.values = append(s.values, v)
	}
	return s
}

// Delete 删除数据
func (s *QueenSession) Delete() (int64, error) {
	// delete from table where id=?
	query := fmt.Sprintf("delete from %s ", s.tableName)
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
		s.db.logger.Error(err)
		return 0, err
	}
	r, err := stmt.Exec(s.whereValues...)
	if err != nil {
		s.db.logger.Error(err)
		return 0, err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		s.db.logger.Error(err)
		return 0, err
	}
	s.db.logger.Info(utils.ConcatenatedString([]string{"影响条数：", strconv.Itoa(int(affected))}))
	return affected, err
}

// Select 查询数据
func (s *QueenSession) Select(data any, fields ...string) ([]any, error) {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer {
		s.db.logger.Error(errors.New("data must be pointer"))
		return nil, errors.New("data must be pointer")
	}
	fieldStr := "*"
	if len(fields) > 0 {
		fieldStr = strings.Join(fields, ",")
	}
	query := fmt.Sprintf("select %s from %s ", fieldStr, s.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())

	stmt, err := s.db.db.Prepare(sb.String())
	if err != nil {
		s.db.logger.Error(err)
		return nil, err
	}
	rows, err := stmt.Query(s.whereValues...)
	if err != nil {
		s.db.logger.Error(err)
		return nil, err
	}
	columns, err := rows.Columns() // id user_name age
	if err != nil {
		s.db.logger.Error(err)
		return nil, err
	}
	result := make([]any, 0)
	for {
		if rows.Next() {
			data := reflect.New(t.Elem()).Interface() // 每次查询的时候data都需要重新换一个地址
			values := make([]any, len(columns))
			fieldScan := make([]any, len(columns))
			for i := range fieldScan {
				fieldScan[i] = &values[i]
			}
			err := rows.Scan(fieldScan...)
			if err != nil {
				s.db.logger.Error(err)
				return nil, err
			}
			tVar := t.Elem()
			vVar := reflect.ValueOf(data).Elem()
			for i := 0; i < tVar.NumField(); i++ {
				name := tVar.Field(i).Name
				tag := tVar.Field(i).Tag
				// id,auto
				sqlTag := tag.Get("qorm")
				if sqlTag == "" {
					sqlTag = strings.ToLower(Name(name))
				} else {
					if strings.Contains(sqlTag, ",") {
						sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
					}
				}

				for j, colName := range columns {
					if sqlTag == colName {
						target := values[j]
						targetValue := reflect.ValueOf(target)
						fieldType := tVar.Field(i).Type
						result := reflect.ValueOf(targetValue.Interface()).Convert(fieldType) // 类型不匹配，需要转换类型
						vVar.Field(i).Set(result)
					}
				}
			}
			result = append(result, data)
		} else {
			break
		}
	}
	return result, nil
}

// SelectOne 只返回一条符合查询条件的数据
func (s *QueenSession) SelectOne(data any, fields ...string) error {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer {
		s.db.logger.Error(errors.New("data must be pointer"))
		return errors.New("data must be pointer")
	}
	fieldStr := "*"
	if len(fields) > 0 {
		fieldStr = strings.Join(fields, ",")
	}
	query := fmt.Sprintf("select %s from %s ", fieldStr, s.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())

	stmt, err := s.db.db.Prepare(sb.String())
	if err != nil {
		s.db.logger.Error(err)
		return err
	}
	rows, err := stmt.Query(s.whereValues...)
	if err != nil {
		s.db.logger.Error(err)
		return err
	}
	// id user_name age
	columns, err := rows.Columns()
	if err != nil {
		s.db.logger.Error(err)
		return err
	}
	values := make([]any, len(columns))
	fieldScan := make([]any, len(columns))
	for i := range fieldScan {
		fieldScan[i] = &values[i]
	}
	if rows.Next() {
		err := rows.Scan(fieldScan...)
		if err != nil {
			s.db.logger.Error(err)
			return err
		}
		tVar := t.Elem()
		vVar := reflect.ValueOf(data).Elem()
		for i := 0; i < tVar.NumField(); i++ {
			name := tVar.Field(i).Name
			tag := tVar.Field(i).Tag
			// id,auto
			sqlTag := tag.Get("qorm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(Name(name))
			} else {
				if strings.Contains(sqlTag, ",") {
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
				}
			}

			for j, colName := range columns {
				if sqlTag == colName {
					target := values[j]
					targetValue := reflect.ValueOf(target)
					fieldType := tVar.Field(i).Type
					result := reflect.ValueOf(targetValue.Interface()).Convert(fieldType)
					vVar.Field(i).Set(result)
				}
			}
		}
	}
	return nil
}

// Count count(*)操作
func (s *QueenSession) Count() (int64, error) {
	return s.Aggregate("count", "*")
}

// Aggregate 计数操作
func (s *QueenSession) Aggregate(funcName string, field string) (int64, error) {
	var fieldSb strings.Builder
	fieldSb.WriteString(funcName)
	fieldSb.WriteString("(")
	fieldSb.WriteString(field)
	fieldSb.WriteString(")")
	query := fmt.Sprintf("select %s from %s ", fieldSb.String(), s.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())

	stmt, err := s.db.db.Prepare(sb.String())
	if err != nil {
		s.db.logger.Error(err)
		return 0, err
	}
	row := stmt.QueryRow(s.whereValues...)
	if row.Err() != nil {
		s.db.logger.Error(err)
		return 0, err
	}
	var result int64
	err = row.Scan(&result)
	if err != nil {
		s.db.logger.Error(err)
		return 0, err
	}
	return result, nil
}

// Where 处理含有where的session
func (s *QueenSession) Where(field string, value any) *QueenSession {
	// id=1 and name=xx
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" = ")
	s.whereParam.WriteString(" ? ")
	s.whereValues = append(s.whereValues, value)
	return s
}

// And 拼接and
func (s *QueenSession) And() *QueenSession {
	s.whereParam.WriteString(" and ")
	return s
}

// Or 拼接or
func (s *QueenSession) Or() *QueenSession {
	s.whereParam.WriteString(" or ")
	return s
}

// Like name like %s%
func (s *QueenSession) Like(field string, value any) *QueenSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" like ")
	s.whereParam.WriteString(" ? ")
	s.whereValues = append(s.whereValues, "%"+value.(string)+"%")
	return s
}

// LikeRight name like s%
func (s *QueenSession) LikeRight(field string, value any) *QueenSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" like ")
	s.whereParam.WriteString(" ? ")
	s.whereValues = append(s.whereValues, value.(string)+"%")
	return s
}

// LikeLeft name like %s
func (s *QueenSession) LikeLeft(field string, value any) *QueenSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" like ")
	s.whereParam.WriteString(" ? ")
	s.whereValues = append(s.whereValues, "%"+value.(string))
	return s
}

// Group group by aa,bb
func (s *QueenSession) Group(field ...string) *QueenSession {
	s.whereParam.WriteString(" group by ")
	s.whereParam.WriteString(strings.Join(field, ","))
	return s
}

// Order  Order("aa","desc","bb","asc)
func (s *QueenSession) Order(field ...string) *QueenSession {
	if len(field)%2 != 0 {
		panic("field num not true")
	}
	s.whereParam.WriteString(" order by ")
	for index, v := range field {
		s.whereParam.WriteString(v + " ")
		if index%2 != 0 && index < len(field)-1 {
			s.whereParam.WriteString(",")
		}
	}
	return s
}

// OrderDesc order by aa,bb desc
func (s *QueenSession) OrderDesc(field ...string) *QueenSession {
	s.whereParam.WriteString(" order by ")
	s.whereParam.WriteString(strings.Join(field, ","))
	s.whereParam.WriteString(" desc ")
	return s
}

// OrderAsc order by aa,bb asc
func (s *QueenSession) OrderAsc(field ...string) *QueenSession {
	s.whereParam.WriteString(" order by ")
	s.whereParam.WriteString(strings.Join(field, ","))
	s.whereParam.WriteString(" asc ")
	return s
}

// Exec 原生SQL支持
func (s *QueenSession) Exec(query string, values ...any) (int64, error) {
	var stmt *sql.Stmt
	var err error
	if s.beginTx {
		stmt, err = s.tx.Prepare(query)
	} else {
		stmt, err = s.db.db.Prepare(query)
	}
	if err != nil {
		return 0, err
	}
	r, err := stmt.Exec(values...)
	if err != nil {
		return 0, err
	}
	if strings.Contains(strings.ToLower(query), "insert") {
		return r.LastInsertId()
	}
	return r.RowsAffected()
}

// QueryRow 原生SQL支持(查询单行数据并将结果映射到结构体中)
func (s *QueenSession) QueryRow(sql string, data any, queryValues ...any) error {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer {
		return errors.New("data must be pointer")
	}
	stmt, err := s.db.db.Prepare(sql)
	if err != nil {
		return err
	}
	rows, err := stmt.Query(queryValues...)
	if err != nil {
		return err
	}
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	values := make([]any, len(columns))
	fieldScan := make([]any, len(columns))
	for i := range fieldScan {
		fieldScan[i] = &values[i]
	}
	if rows.Next() {
		err := rows.Scan(fieldScan...)
		if err != nil {
			return err
		}
		tVar := t.Elem()
		vVar := reflect.ValueOf(data).Elem()
		for i := 0; i < tVar.NumField(); i++ {
			name := tVar.Field(i).Name
			tag := tVar.Field(i).Tag
			sqlTag := tag.Get("qorm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(Name(name))
			} else {
				if strings.Contains(sqlTag, ",") {
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
				}
			}
			for j, colName := range columns {
				if sqlTag == colName {
					target := values[j]
					targetValue := reflect.ValueOf(target)
					fieldType := tVar.Field(i).Type
					result := reflect.ValueOf(targetValue.Interface()).Convert(fieldType) // 类型不匹配，进行类型转换
					vVar.Field(i).Set(result)
				}
			}
		}
	}
	return nil
}

// Begin 开启事务
func (s *QueenSession) Begin() error {
	tx, err := s.db.db.Begin()
	if err != nil {
		return err
	}
	s.tx = tx
	s.beginTx = true
	return nil
}

// Commit 提交事务
func (s *QueenSession) Commit() error {
	err := s.tx.Commit()
	if err != nil {
		return err
	}
	s.beginTx = false
	return nil
}

// Rollback 事务回滚
func (s *QueenSession) Rollback() error {
	err := s.tx.Rollback()
	if err != nil {
		return err
	}
	s.beginTx = false
	return nil
}
