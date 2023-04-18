package service

import "github.com/qingbo1011/qiaomu/rpc"

type GoodsService struct {
	Find func(args map[string]any) ([]byte, error) `qrpc:"GET,/goods/find"`
}

func (*GoodsService) Env() rpc.HttpConfig {
	return rpc.HttpConfig{
		Host: "127.0.0.1",
		Port: 8082,
	}
}
