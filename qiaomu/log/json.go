package qlog

import (
	"encoding/json"
	"fmt"
	"time"
)

// JsonFormatter JSON格式
type JsonFormatter struct {
	TimeDisplay bool // 是否显示时间
}

// Format 日志格式化(JSON格式)
func (f *JsonFormatter) Format(param *LoggingFormatParam) string {
	if param.LoggerFields == nil {
		param.LoggerFields = make(Fields)
	}

	now := time.Now()
	if f.TimeDisplay {
		param.LoggerFields["log_time"] = now.Format("2006/01/02 - 15:04:05")
	}
	param.LoggerFields["msg"] = param.Msg
	param.LoggerFields["log_level"] = param.Level.Level()
	marshal, err := json.Marshal(param.LoggerFields)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s", string(marshal))
}
