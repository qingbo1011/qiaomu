package main

import (
	"github.com/qingbo1011/qiaomu"
)

func main() {
	engine := qiaomu.Default()
	group := engine.Group("order")
	group.Get("/find", func(ctx *qiaomu.Context) {
		// 通过商品中心查询商品的信息
		// http的方式进行调用

	})
	engine.Run(":8082")
}
