package qiaomu

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	greenBg   = "\033[97;42m"
	whiteBg   = "\033[90;47m"
	yellowBg  = "\033[90;43m"
	redBg     = "\033[97;41m"
	blueBg    = "\033[97;44m"
	magentaBg = "\033[97;45m"
	cyanBg    = "\033[97;46m"
	green     = "\033[32m"
	white     = "\033[37m"
	yellow    = "\033[33m"
	red       = "\033[31m"
	blue      = "\033[34m"
	magenta   = "\033[35m"
	cyan      = "\033[36m"
	reset     = "\033[0m"
)

var DefaultWriter io.Writer = os.Stdout

// LoggingConfig 日志配置
type LoggingConfig struct {
	Formatter LoggerFormatter
	out       io.Writer
	IsColor   bool
}

type LoggerFormatter = func(params *LogFormatterParams) string

// LogFormatterParams 日志格式参数
type LogFormatterParams struct {
	Request        *http.Request
	TimeStamp      time.Time
	StatusCode     int
	Latency        time.Duration
	ClientIP       net.IP
	Method         string
	Path           string
	IsDisplayColor bool
}

func LoggingWithConfig(conf LoggingConfig, next HandlerFunc) HandlerFunc {
	formatter := conf.Formatter
	//if formatter == nil {
	//	formatter = defaultFormatter
	//}
	out := conf.out
	displayColor := false
	if out == nil {
		out = DefaultWriter
		displayColor = true
	}
	return func(ctx *Context) {
		r := ctx.R
		param := &LogFormatterParams{
			Request:        r,
			IsDisplayColor: displayColor,
		}
		// Start timer
		start := time.Now()
		path := r.URL.Path
		raw := r.URL.RawQuery
		next(ctx)
		stop := time.Now()
		latency := stop.Sub(start)
		ip, _, _ := net.SplitHostPort(strings.TrimSpace(ctx.R.RemoteAddr))
		clientIP := net.ParseIP(ip)
		method := r.Method
		statusCode := ctx.StatusCode

		if raw != "" {
			path = path + "?" + raw
		}

		param.TimeStamp = stop
		param.StatusCode = statusCode
		param.Latency = latency
		param.Path = path
		param.ClientIP = clientIP
		param.Method = method

		fmt.Fprint(out, formatter(param))
	}
}
