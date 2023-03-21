package main

import (
	"fmt"
	"github.com/qingbo1011/qiaomu"
	"net/http"
)

func main() {
	engine := qiaomu.New()
	group := engine.Group("user")
	group.Add("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello,user")
	})
	engine.Run()
}
