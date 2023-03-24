package main

import (
	"fmt"
	"github.com/qingbo1011/qiaomu"
)

func main() {
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
}
