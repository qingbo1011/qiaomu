package qiaomu

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	tracer2 "github.com/qingbo1011/qiaomu/tracer"
	"github.com/uber/jaeger-client-go/config"
)

// Tracer 链路追踪中间件
func Tracer(serviceName string, samplerConfig *config.SamplerConfig, reporter *config.ReporterConfig, options ...config.Option) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			// 接收jaeger的信息 解析上下文
			// 使用 opentracing.GlobalTracer() 获取全局 Tracer
			tracer, closer, spanContext, _ := tracer2.CreateTracerHeader(serviceName, ctx.R.Header, samplerConfig, reporter, options...)
			defer closer.Close()
			//
			// 生成依赖关系，并新建一个 span
			// 生成了References与[]SpanReference的依赖关系
			startSpan := tracer.StartSpan(ctx.R.URL.Path, ext.RPCServerOption(spanContext))
			defer startSpan.Finish()
			// 记录tag
			// 记录请求Url
			ext.HTTPUrl.Set(startSpan, ctx.R.URL.Path)
			// Http Method
			ext.HTTPMethod.Set(startSpan, ctx.R.Method)
			// 记录组件名称
			ext.Component.Set(startSpan, "Msgo-Http")

			// 在header中加上当前进程的上下文信息
			ctx.R = ctx.R.WithContext(opentracing.ContextWithSpan(ctx.R.Context(), startSpan))
			next(ctx)
			// 继续设置tag
			ext.HTTPStatusCode.Set(startSpan, uint16(ctx.StatusCode))
		}
	}
}
