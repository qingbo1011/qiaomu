package main

import (
	"fmt"
	"net/http"

	"github.com/qingbo1011/qiaomu"
)

func main() {
	engine := qiaomu.New()
	engine.Add("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "qiaomu test")
	})
	engine.Run()
}
