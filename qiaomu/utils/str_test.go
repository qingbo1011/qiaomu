package utils

import (
	"testing"
)

func TestConcatenatedString(t *testing.T) {
	var testcases = []struct {
		in  []string
		out string
	}{
		{[]string{"/", "user", "/", "info"}, "/user/info"},
		{[]string{"/", "hello", "/", "info"}, "/hello/info"},
	}
	for _, testcase := range testcases {
		if ConcatenatedString(testcase.in) != testcase.out {
			t.Errorf("got %q, want %q", ConcatenatedString(testcase.in), testcase.out)
		}
	}
}
