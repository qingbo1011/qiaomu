package qiaomu

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/qingbo1011/qiaomu/qerror"
)

// Recovery 错误处理中间件
func Recovery(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		defer func() {
			if err := recover(); err != nil {
				err2 := err.(error)
				if err2 != nil {
					var qError *qerror.QError
					if errors.As(err2, &qError) {
						qError.ExecResult()
						return
					}
				}
				ctx.Logger.Error(detailMsg(err))
				ctx.Fail(http.StatusInternalServerError, "Internal Server Error!")
			}
		}()
		next(ctx)
	}
}

// 使用runtime包拼接出错误代码位置
func detailMsg(err any) string {
	var pcs [32]uintptr
	n := runtime.Callers(0, pcs[:])
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%v\n", err))
	for _, pc := range pcs[0:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		sb.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
	}
	return sb.String()
}
