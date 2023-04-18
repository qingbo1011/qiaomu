package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/qingbo1011/goodscenter/api"
	"github.com/qingbo1011/goodscenter/model"
	"github.com/qingbo1011/ordercenter/service"
	"github.com/qingbo1011/qiaomu"
	"github.com/qingbo1011/qiaomu/rpc"
	"github.com/qingbo1011/qiaomu/tracer"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

func main() {
	engine := qiaomu.Default()
	client := rpc.NewHttpClient()
	client.RegisterHttpService("goods", &service.GoodsService{})
	createTracer, closer, err := tracer.CreateTracer("orderCenter",
		&config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		&config.ReporterConfig{
			LogSpans:          true,
			CollectorEndpoint: "http://192.168.200.100:14268/api/traces",
		}, config.Logger(jaeger.StdLogger),
	)
	if err != nil {
		log.Println(err)
	}
	defer closer.Close()

	group := engine.Group("order")
	group.Get("/findhttp", func(ctx *qiaomu.Context) {
		// 通过商品中心查询商品的信息
		// http的方式进行调用
		params := make(map[string]any)
		params["id"] = ctx.GetQuery("id")
		params["name"] = "qiaomu"
		span := createTracer.StartSpan("find")
		defer span.Finish()
		session := client.Session()
		session.ReqHandler = func(req *http.Request) {
			ext.SpanKindRPCClient.Set(span)
			createTracer.Inject(span.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header))
		}
		body, err := session.Do("goods", "Find").(*service.GoodsService).Find(params)
		if err != nil {
			panic(err)
		}
		v := &model.Result{}
		json.Unmarshal(body, v)
		ctx.JSON(http.StatusOK, v)
	})
	group.Get("/findgrpc", func(ctx *qiaomu.Context) {
		config := rpc.DefaultGrpcClientConfig()
		config.Address = "127.0.0.1:9111"
		client, _ := rpc.NewGrpcClient(config)
		defer client.Conn.Close()
		goodsApiClient := api.NewGoodsApiClient(client.Conn)
		goodsResponse, _ := goodsApiClient.Find(context.Background(), &api.GoodsRequest{})
		ctx.JSON(http.StatusOK, goodsResponse)
	})
	engine.Run(":8083")
}
