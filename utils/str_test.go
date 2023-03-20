package utils

import (
	"fmt"
	"testing"
)

func TestConcatenatedString(t *testing.T) {
	s := []string{"/", "user", "/", "info"}
	str := ConcatenatedString(s)
	fmt.Println(str)
}
