package main

import (
	"encoding/gob"
	"log"
	"net/http"
	"time"

	"github.com/qingbo1011/goodscenter/api"
	"github.com/qingbo1011/goodscenter/model"
	"github.com/qingbo1011/goodscenter/service"
	"github.com/qingbo1011/qiaomu"
	"github.com/qingbo1011/qiaomu/register"
	"github.com/qingbo1011/qiaomu/rpc"
	"google.golang.org/grpc"
)

func main() {
	engine := qiaomu.Default()
	group := engine.Group("goods")
	group.Get("/find", func(ctx *qiaomu.Context) {
		goods := &model.Goods{Id: 1000, Name: "8082的商品"}
		ctx.JSON(http.StatusOK, &model.Result{Code: 200, Msg: "success", Data: goods})
	})
	//
	server, _ := rpc.NewGrpcServer(":9111")
	server.Register(func(g *grpc.Server) {
		api.RegisterGoodsApiServer(g, &api.GoodsRpcService{})
	})
	err := server.Run()
	log.Println(err)
	tcpServer, err := rpc.NewTcpServer("127.0.0.1", 9222)
	tcpServer.SetRegister("etcd", register.Option{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
		Host:        "127.0.0.1",
		Port:        9222,
	})
	log.Println(err)
	gob.Register(&model.Result{})
	gob.Register(&model.Goods{})
	tcpServer.Register("goods", &service.GoodsRpcService{})
	tcpServer.LimiterTimeOut = time.Second
	tcpServer.SetLimiter(10, 100)
	tcpServer.Run()
	cli := register.QueenEtcdRegister{}
	cli.CreateCli(register.Option{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	})
	cli.RegisterService("goodsCenter", "127.0.0.1", 8082)
	engine.Run(":8082")
}
