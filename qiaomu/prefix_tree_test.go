package qiaomu

import (
	"fmt"
	"testing"
)

func TestTreeNode(t *testing.T) {
	root := &treeNode{name: "/", children: make([]*treeNode, 0)}

	root.Put("/user/get/:id")
	root.Put("/user/create/hello")
	root.Put("/user/create/aaa")
	root.Put("/order/get/aaa")

	node := root.Get("/user/get/1")
	fmt.Println(node) // &{:id [] /user/get/:id}
	node = root.Get("/user/create/hello")
	fmt.Println(node) // &{hello [] /user/create/hello}
	node = root.Get("/user/create/aaa")
	fmt.Println(node) // &{aaa [] /user/create/aaa}
	node = root.Get("/order/get/aaa")
	fmt.Println(node) // &{aaa [] /order/get/aaa}
}
