package main

import (
	"fmt"
	"github.com/qingbo1011/qiaomu"
)

// 路由测试
/*func main() {
	engine := qiaomu.New()
	group := engine.Group("user")
	group.Get("/hello", func(ctx *qiaomu.Context) {
		fmt.Fprint(ctx.W, "Get,/hello")
	})
	group.Post("/hello", func(ctx *qiaomu.Context) {
		fmt.Fprint(ctx.W, "Post,/hello")
	})
	group.Any("/info", func(ctx *qiaomu.Context) {
		fmt.Fprint(ctx.W, "Any,/info")
	})

	group.Get("/getid/:id", func(ctx *qiaomu.Context) {
		fmt.Fprint(ctx.W, "Get,/getid/:id")
	})
	group.Get("/blog/look", func(ctx *qiaomu.Context) {
		fmt.Fprint(ctx.W, "Get,/blog/look")
	})
	group.Post("/log/*", func(ctx *qiaomu.Context) {
		fmt.Fprint(ctx.W, "Post,/log/*")
	})
	engine.Run()
}*/

// Log 定义一个测试中间件
func Log(next qiaomu.HandlerFunc) qiaomu.HandlerFunc {
	return func(ctx *qiaomu.Context) {
		fmt.Println("打印请求参数")
		next(ctx)
		fmt.Println("返回执行时间")
	}
}

// 中间件测试
func main() {
	engine := qiaomu.New()
	group := engine.Group("user")
	// 具体路由使用中间件
	group.Get("/hello/get", func(ctx *qiaomu.Context) {
		fmt.Println("handler")
		fmt.Fprint(ctx.W, "/hello/get GET")
	}, Log)
	group.Get("/hello/get2", func(ctx *qiaomu.Context) {
		fmt.Println("handler")
		fmt.Fprint(ctx.W, "/hello/get2 GET")
	})

	// 整个路由组使用中间件
	group2 := engine.Group("student")
	group2.Use(Log)
	group2.Get("/info", func(ctx *qiaomu.Context) {
		fmt.Println("handler")
		fmt.Fprint(ctx.W, "/info GET")
	})
	group2.Get("/info2", func(ctx *qiaomu.Context) {
		fmt.Println("handler")
		fmt.Fprint(ctx.W, "/info2 GET")
	})

	engine.Run()
}