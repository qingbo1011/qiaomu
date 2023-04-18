package service

import "github.com/qingbo1011/goodscenter/model"

type GoodsRpcService struct {
}

func (*GoodsRpcService) Find(id int64) *model.Result {
	goods := model.Goods{Id: 1000, Name: "商品中心8082商品"}
	return &model.Result{Code: 200, Msg: "success", Data: goods}
}
