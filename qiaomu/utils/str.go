package utils

import "strings"

// ConcatenatedString 高性能拼接字符串
func ConcatenatedString(s []string) string {
	var builder strings.Builder
	for _, str := range s {
		builder.WriteString(str)
	}
	return builder.String()
}
