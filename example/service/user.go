package service

import (
	"fmt"
	"net/url"

	_ "github.com/go-sql-driver/mysql"
	"github.com/qingbo1011/qiaomu/orm"
)

// Tag写法示例
//type User struct {
//	Id       int64  `qorm:"id,auto_increment"`
//	UserName string `qorm:"user_name"`
//	Password string `qorm:"password"`
//	Age      int    `qorm:"age"`
//}

type User struct {
	Id       int64
	UserName string `qorm:"user_name"`
	Password string
	Age      int
}

func SaveUser() {
	dataSourceName := fmt.Sprintf("root:1234@tcp(43.138.57.192:3310)/qiaomu?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	db.Prefix = "queen_" // 数据库表前缀
	user := &User{
		UserName: "test",
		Password: "123456",
		Age:      21,
	}
	_, _, err := db.New(&User{}).Insert(user)
	if err != nil {
		panic(err)
	}

	db.Close()
}

func SaveUserBatch() {
	dataSourceName := fmt.Sprintf("root:1234@tcp(43.138.57.192:3310)/qiaomu?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	db.Prefix = "queen_" // 数据库表前缀
	user := &User{
		UserName: "test1",
		Password: "12345612",
		Age:      18,
	}
	user1 := &User{
		UserName: "test2",
		Password: "123456111",
		Age:      22,
	}
	var users []any
	users = append(users, user, user1)
	_, _, err := db.New(&User{}).InsertBatch(users)
	if err != nil {
		panic(err)
	}
	db.Close()
}

func UpdateUser() {
	dataSourceName := fmt.Sprintf("root:1234@tcp(43.138.57.192:3310)/qiaomu?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	db.Prefix = "queen_" // 数据库表前缀
	//id, _, err := db.New().Where("id", 1006).Where("age", 54).Update(user)
	// 单个插入
	user := &User{
		UserName: "queen",
		Password: "123456",
		Age:      30,
	}
	id, _, err := db.New(&User{}).Insert(user)
	if err != nil {
		panic(err)
	}
	fmt.Println(id)

	// 批量插入
	var users []any
	users = append(users, user)
	id, _, err = db.New(&User{}).InsertBatch(users)
	if err != nil {
		panic(err)
	}
	fmt.Println(id)
	// 更新
	id, _, err = db.
		New(&User{}).
		Where("id", 1006).
		UpdateParam("age", 100).
		Update()
	//// 查询单行数据
	//err = db.New(&User{}).
	//	Where("id", 1006).
	//	Or().
	//	Where("age", 30).
	//	SelectOne(user, "user_name")
	//// 查询多行数据
	//users, err = db.New(&User{}).Select(&User{})
	//if err != nil {
	//	panic(err)
	//}
	for _, v := range users {
		u := v.(*User)
		fmt.Println(u)
	}

	if err != nil {
		panic(err)
	}
	fmt.Println(id)

	db.Close()
}

func DeleteOne() {
	dataSourceName := fmt.Sprintf("root:1234@tcp(43.138.57.192:3310)/qiaomu?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	db.Prefix = "queen_" // 数据库表前缀
	user := &User{}
	_, err := db.New(user).Where("user_name", "test2").Delete()
	if err != nil {
		panic(err)
	}
	db.Close()
}

func SelectOne() {
	dataSourceName := fmt.Sprintf("root:1234@tcp(43.138.57.192:3310)/qiaomu?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	db.Prefix = "queen_" // 数据库表前缀
	user := &User{}
	err := db.New(user).
		Where("id", 1006).
		Or().
		Where("age", 100).
		SelectOne(user, "user_name", "password", "age")
	if err != nil {
		panic(err)
	}
	fmt.Println(user)
	db.Close()
}

func Select() {
	dataSourceName := fmt.Sprintf("root:1234@tcp(43.138.57.192:3310)/qiaomu?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	db.Prefix = "queen_" // 数据库表前缀
	user := &User{}
	//users, err := db.New(user).Where("user_name", "queen").Order("id", "asc", "age", "desc").Select(user)
	users, err := db.New(user).Where("user_name", "queen").OrderAsc("age").Select(user)
	if err != nil {
		panic(err)
	}
	for _, v := range users {
		u := v.(*User)
		fmt.Println(u)
	}
	db.Close()
}

func Count() {
	dataSourceName := fmt.Sprintf("root:1234@tcp(43.138.57.192:3310)/qiaomu?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	db.Prefix = "queen_" // 数据库表前缀
	user := &User{}
	count, err := db.New(user).Count()
	if err != nil {
		panic(err)
	}
	fmt.Println(count)
	db.Close()
}

func Exec() {
	dataSourceName := fmt.Sprintf("root:1234@tcp(43.138.57.192:3310)/qiaomu?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	db.Prefix = "queen_" // 数据库表前缀
	user := &User{}
	//// 示例1：执行插入操作
	//// 执行INSERT语句，返回最后插入的自增ID
	//lastInsertId, err := db.New(user).Exec("INSERT INTO queen_user (user_name, age) VALUES (?, ?)", "qingyu", 19) // []any切片要展开！
	//if err != nil {
	//	fmt.Printf("Error: %v\n", err)
	//} else {
	//	fmt.Printf("Inserted user with ID: %d\n", lastInsertId)
	//}
	// 示例2：执行更新操作
	//query := "UPDATE queen_user SET user_name = ? WHERE id = ?"
	//newName := "qiaomu111"
	//affectedRows, err := db.New(user).Exec(query, newName, 1001)
	//if err != nil {
	//	fmt.Printf("Error: %v\n", err)
	//} else {
	//	fmt.Printf("Updated %d rows\n", affectedRows)
	//}
	// 示例3：执行删除操作
	query := "DELETE FROM queen_user WHERE user_name = ?"
	username := "test1"
	affectedRows, err := db.New(user).Exec(query, username)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Deleted %d rows\n", affectedRows)
	}
}

func QueryRow() {
	dataSourceName := fmt.Sprintf("root:1234@tcp(43.138.57.192:3310)/qiaomu?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	db.Prefix = "queen_" // 数据库表前缀
	user := &User{}
	err := db.New(user).QueryRow("SELECT * FROM queen_user WHERE id = ?", user, 1013)
	if err != nil {
		panic(err)
	}
	fmt.Println(user)
}

func Transaction() {
	dataSourceName := fmt.Sprintf("root:1234@tcp(43.138.57.192:3310)/qiaomu?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	db.Prefix = "queen_" // 数据库表前缀
	user := &User{}
	session := db.New(user)
	// 开启事务
	err := session.Begin()
	if err != nil {
		panic(err)
	}
	// 开始一些列SQL操作
	user1 := &User{
		UserName: "test",
		Password: "123456",
		Age:      21,
	}
	_, _, err = session.Insert(user1)
	if err != nil {
		panic(err)
	}
	_, err = session.Exec("UPDATE queen_user SET user_name = ? WHERE id = ?", "rick", 1002)
	if err != nil {
		panic(err)
	}
	//// 提交事务
	//err = session.Commit()
	//if err != nil {
	//	panic(err)
	//}
	// 回滚事务
	err = session.Rollback()
	if err != nil {
		panic(err)
	}
}
