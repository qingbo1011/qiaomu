package utils

import (
	"fmt"
	"reflect"
	"strings"
)

// ConcatenatedString 高性能拼接字符串
func ConcatenatedString(s []string) string {
	var builder strings.Builder
	for _, str := range s {
		builder.WriteString(str)
	}
	return builder.String()
}

// SubStringLast 返回str中substr(若有多个部分取最前的索引)后的所有字符串。（例：str := "/user/info/user" substr := "/user" 返回/info/user）
func SubStringLast(str string, substr string) string {
	index := strings.Index(str, substr)
	if index == -1 { // 先查找有没有
		return ""
	}
	return str[index+len(substr):]
}

func JoinStrings(data ...any) string {
	var sb strings.Builder
	for _, v := range data {
		sb.WriteString(check(v))
	}
	return sb.String()
}

func check(v any) string {
	value := reflect.ValueOf(v)
	switch value.Kind() {
	case reflect.String:
		return v.(string)
	default:
		return fmt.Sprintf("%v", v)
	}
}
