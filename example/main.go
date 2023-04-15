package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/qingbo1011/qiaomu"
	"github.com/qingbo1011/qiaomu/bind"
	qlog "github.com/qingbo1011/qiaomu/log"
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
	Name      string   `xml:"name" json:"name"`
	Age       int      `xml:"age" json:"age" validate:"required,max=50,min=18"`
	Addresses []string `json:"addresses"`
	Email     string   `json:"email"`
}

// 页面渲染(模板支持)测试
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
/*func main() {
	engine := qiaomu.New()
	group := engine.Group("user")

	group.Get("/add", func(ctx *qiaomu.Context) {
		name := ctx.GetQuery("name")
		fmt.Printf("name: %v , ok: %v \n", name, true)
	})
	group.Get("/adds", func(ctx *qiaomu.Context) {
		querys, ok := ctx.GetQueryArray("id")
		fmt.Println("参数切片: ", querys, " , ok: ", ok)
	})
	group.Get("/default", func(ctx *qiaomu.Context) {
		name := ctx.GetDefaultQuery("name", "张三")
		fmt.Printf("name: %v , ok: %v \n", name, true)
	})
	group.Get("/queryMap", func(ctx *qiaomu.Context) {
		m, _ := ctx.GetQueryMap("user")
		ctx.JSON(http.StatusOK, m)
	})
	group.Post("/formPost", func(ctx *qiaomu.Context) {
		m, _ := ctx.GetPostFormMap("user")
		files := ctx.FormFiles("file")
		for _, file := range files {
			ctx.SaveUploadedFile(file, "./upload/"+file.Filename)
		}
		ctx.JSON(http.StatusOK, m)
	})
	group.Post("/jsonParam", func(ctx *qiaomu.Context) {
		user := User{}
		ctx.DisallowUnknownFields = true
		//ctx.IsValidate = true
		err := ctx.ShouldBind(&user, bind.JSON)
		if err == nil {
			ctx.JSON(http.StatusOK, user)
		} else {
			log.Println(err)
		}
	})
	group.Post("/xmlParam", func(ctx *qiaomu.Context) {
		user := User{}
		ctx.DisallowUnknownFields = true
		//ctx.IsValidate = true
		err := ctx.BindXML(&user)
		if err == nil {
			ctx.JSON(http.StatusOK, user)
		} else {
			log.Println(err)
		}
	})
	engine.Run()
}*/

// 日志处理测试
/*func main() {
	engine := qiaomu.New()
	//engine.Logger = qlog.New()
	//engine.Logger.Level = qlog.LevelDebug // 设置日志级别
	group := engine.Group("user")
	//group.Use(qiaomu.Logging)	// 注册中间件方式
	group.Post("/jsonParam", func(ctx *qiaomu.Context) {
		user := User{}
		ctx.DisallowUnknownFields = true
		ctx.Logger = qlog.Default()
		ctx.Logger.Level = qlog.LevelDebug                            // 设置日志级别
		ctx.Logger.Formatter = &qlog.JsonFormatter{TimeDisplay: true} // JSON格式
		ctx.Logger.SetLogPath("./log")                                // 日志持久化存储一般使用JSON格式
		//ctx.IsValidate = true
		ctx.Logger.WithFields(qlog.Fields{
			"name": "qiaomu",
			"id":   1000,
		}).Debug("debug日志")
		ctx.Logger.Info("info日志")
		ctx.Logger.Error("error日志")
		err := ctx.ShouldBind(&user, bind.JSON)
		if err == nil {
			ctx.JSON(http.StatusOK, user)
		} else {
			log.Println(err)
		}
	})
	engine.Run()

}*/

// 错误处理测试
func main() {
	engine := qiaomu.Default()
	//engine.RegisterErrorHandler(func(err error) (int, any) {
	//	switch e := err.(type) {
	//	case *BlogError:
	//		return http.StatusOK, e.Response()
	//	default:
	//		return http.StatusInternalServerError, "Internal Server Error"
	//	}
	//})
	group := engine.Group("user")
	var u *User
	group.Post("/jsonParam", func(ctx *qiaomu.Context) {
		user := User{}
		ctx.DisallowUnknownFields = true
		u.Age = 18 // 人为制造panic
		//ctx.IsValidate = true
		ctx.Logger = engine.Logger
		ctx.Logger.WithFields(qlog.Fields{
			"name": "qiaomu",
			"id":   1000,
		}).Debug("debug日志")
		ctx.Logger.Info("info日志")
		ctx.Logger.Error("error日志")
		err := ctx.ShouldBind(&user, bind.JSON)
		if err == nil {
			ctx.JSON(http.StatusOK, user)
		} else {
			log.Println(err)
		}
	})
	engine.Run()
}

// 协程池测试
func main() {
	engine := qiaomu.Default()
	group := engine.Group("user")

	engine.Run()
}
