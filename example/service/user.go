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
	UserName string
	Password string
	Age      int
}

func SaveUser() {
	dataSourceName := fmt.Sprintf("root:1234@tcp(43.138.57.192:3310)/qiaomu?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	db.Prefix = "queen_" // 数据库前缀
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
	db.Prefix = "queen_" // 数据库前缀
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
	db.Prefix = "queen_" // 数据库前缀
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