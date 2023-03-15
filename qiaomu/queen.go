package qiaomu

import (
	"fmt"
	"log"
	"net/http"
)

type Engine struct {
}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) Run() {
	fmt.Println("8081端口运行中")
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		log.Fatalln(err)
	}
}
