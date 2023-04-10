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
/*func main() {
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
}*/

type User struct {
	Name string `xml:"name"`
	Age  int    `xml:"age"`
}

// 页面渲染（模板支持）测试
/*func main() {
	engine := qiaomu.New()
	group := engine.Group("user")
	group.Get("/html", func(ctx *qiaomu.Context) {
		ctx.HTML(http.StatusOK, "<h1>乔木")
	})
	user := &User{
		Name: "李四",
		Age:  18,
	}
	group.Get("/htmlTemplate", func(ctx *qiaomu.Context) {
		err := ctx.HTMLTemplate("login.html", user, "template/login.html", "template/header.html")
		if err != nil {
			log.Println(err)
		}
	})
	group.Get("/htmlTemplateGlob", func(ctx *qiaomu.Context) {
		ctx.HTMLTemplateGlob("login.html", user, "template/*.html")
	})

	engine.LoadTemplate("template/*.html")
	group.Get("/template", func(ctx *qiaomu.Context) {
		ctx.Template("login.html", user)
	})
	group.Get("/json", func(ctx *qiaomu.Context) {
		ctx.JSON(http.StatusOK, user)
	})
	group.Get("/xml", func(ctx *qiaomu.Context) {
		err := ctx.XML(http.StatusOK, user)
		if err != nil {
			log.Println(err)
		}
	})
	group.Get("/csv", func(ctx *qiaomu.Context) {
		ctx.File("template/file_test.csv")
	})
	group.Get("/csvname", func(ctx *qiaomu.Context) {
		ctx.FileAttachment("template/file_test.csv", "queen.csv")
	})
	group.Get("/fs", func(ctx *qiaomu.Context) {
		ctx.FileFromFS("file_test.csv", http.Dir("template"))
	})
	group.Get("/redirect", func(ctx *qiaomu.Context) {
		ctx.Redirect(http.StatusFound, "/user/html")
	})
	group.Get("/string", func(ctx *qiaomu.Context) {
		ctx.String(http.StatusFound, "%s是一个%s web框架", "qiaomu", "go的")
	})

	engine.Run()
}*/

// 参数处理测试
func main() {

}
