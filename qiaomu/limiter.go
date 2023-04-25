package qiaomu

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

// Limiter 限流中间件
func Limiter(limit, cap int) MiddlewareFunc {
	li := rate.NewLimiter(rate.Limit(limit), cap)
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			// 实现限流
			con, cancel := context.WithTimeout(context.Background(), time.Duration(1)*time.Second)
			defer cancel()
			err := li.WaitN(con, 1)
			if err != nil {
				ctx.String(http.StatusForbidden, "被限流了")
				return
			}
			next(ctx)
		}
	}
}
