package qiaomu

import (
	"github.com/qingbo1011/qiaomu/utils"
	"strings"
)

// 前缀树节点
type treeNode struct {
	name       string      // 节点名称，比如/user就是user
	children   []*treeNode // 子节点
	routerName string
	isEnd      bool // 表示该节点是否是某一个路由的终点（解决/user/hello/xx 这样的路由 /user/hello 访问这个路径也一样能从前 缀树查找出来，并不会报404的bug）
}

// Put 根据url(/user/get/:id)来设置前缀树
func (t *treeNode) Put(path string) {
	root := t
	strs := strings.Split(path, "/")
	for i, name := range strs {
		if i == 0 { // /user/get/:id为例，切割后的切片，索引为0的数据strs[0]为空字符串 ""
			continue
		}
		children := t.children
		isMatch := false
		for _, node := range children {
			if node.name == name {
				isMatch = true
				t = node
				break
			}
		}
		if !isMatch {
			node := &treeNode{
				name:     name,
				children: make([]*treeNode, 0),
				isEnd:    i == len(strs)-1,
			}
			children = append(children, node)
			t.children = children
			t = node
		}
	}
	t = root
}

// Get 根据url去匹配到前缀树的节点
func (t *treeNode) Get(path string) *treeNode {
	strs := strings.Split(path, "/")
	routerName := ""
	for i, name := range strs {
		if i == 0 {
			continue
		}
		children := t.children
		isMatch := false
		for _, node := range children {
			if node.name == name ||
				node.name == "*" ||
				strings.Contains(node.name, ":") {
				isMatch = true
				routerName = utils.ConcatenatedString([]string{routerName, "/", node.name})
				node.routerName = routerName
				t = node
				if i == len(strs)-1 {
					return node
				}
				break
			}
		}
		if !isMatch {
			for _, node := range children {
				// /user/**
				// /user/get/userInfo // /user/aa/bb
				if node.name == "**" {
					routerName = utils.ConcatenatedString([]string{routerName, "/", node.name})
					node.routerName = routerName
					return node
				}
			}
		}
	}
	return nil
}
