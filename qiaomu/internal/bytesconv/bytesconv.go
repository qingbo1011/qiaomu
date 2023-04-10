package bytesconv

import "unsafe"

// StringToBytes 高性能字符串转字节
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}
