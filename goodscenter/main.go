package main

import (
	"net/http"

	"github.com/qingbo1011/goodscenter/model"
	"github.com/qingbo1011/qiaomu"
)

func main() {
	engine := qiaomu.Default()
	group := engine.Group("goods")
	group.Get("/find", func(ctx *qiaomu.Context) {
		goods := &model.Goods{Id: 1000, Name: "8082的商品"}
		ctx.JSON(http.StatusOK, &model.Result{Code: 200, Msg: "success", Data: goods})
	})
	engine.Run(":8082")
}
