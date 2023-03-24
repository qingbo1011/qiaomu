package qiaomu

import (
	"fmt"
	"github.com/qingbo1011/qiaomu/utils"
	"log"
	"net/http"
)

const (
	MethodAny = "ANY"
)

type HandlerFunc func(ctx *Context)

type Context struct {
	W http.ResponseWriter
	R *http.Request
}

type router struct {
	groups []*routerGroup
}

func (r *router) Group(name string) *routerGroup {
	g := &routerGroup{
		groupName:        name,
		handlerMap:       make(map[string]map[string]HandlerFunc),
		handlerMethodMap: make(map[string][]string),
		treeNode: &treeNode{
			name:     "/",
			children: make([]*treeNode, 0),
		},
	}
	r.groups = append(r.groups, g)
	return g
}

type routerGroup struct {
	groupName        string
	handlerMap       map[string]map[string]HandlerFunc
	handlerMethodMap map[string][]string
	treeNode         *treeNode
}

// Handle method的有效性校验
func (r *routerGroup) Handle(name string, method string, handlerFunc HandlerFunc) {
	r.handle(name, method, handlerFunc)
}

// Any 任意类型的路由
func (r *routerGroup) Any(name string, handleFunc HandlerFunc) {
	r.handle(name, MethodAny, handleFunc)
}

// Get Get类型路由
func (r *routerGroup) Get(name string, handleFunc HandlerFunc) {
	r.handle(name, http.MethodGet, handleFunc)
}

// Head Head类型路由
func (r *routerGroup) Head(name string, handleFunc HandlerFunc) {
	r.handle(name, http.MethodHead, handleFunc)
}

// Post Post类型路由
func (r *routerGroup) Post(name string, handleFunc HandlerFunc) {
	r.handle(name, http.MethodPost, handleFunc)
}

// Put Put类型路由
func (r *routerGroup) Put(name string, handleFunc HandlerFunc) {
	r.handle(name, http.MethodPut, handleFunc)
}

// Patch Patch类型路由
func (r *routerGroup) Patch(name string, handleFunc HandlerFunc) {
	r.handle(name, http.MethodPatch, handleFunc)
}

// Delete Delete类型路由
func (r *routerGroup) Delete(name string, handleFunc HandlerFunc) {
	r.handle(name, http.MethodDelete, handleFunc)
}

// Connect Connect类型路由
func (r *routerGroup) Connect(name string, handleFunc HandlerFunc) {
	r.handle(name, http.MethodConnect, handleFunc)
}

// Options Options类型路由
func (r *routerGroup) Options(name string, handleFunc HandlerFunc) {
	r.handle(name, http.MethodOptions, handleFunc)
}

// Trace Trace类型路由
func (r *routerGroup) Trace(name string, handleFunc HandlerFunc) {
	r.handle(name, http.MethodTrace, handleFunc)
}

// 统一处理
func (r *routerGroup) handle(name string, method string, handlerFunc HandlerFunc) {
	_, ok := r.handlerMap[name]
	if !ok {
		r.handlerMap[name] = make(map[string]HandlerFunc)
	}
	r.handlerMap[name][method] = handlerFunc
	r.handlerMethodMap[method] = append(r.handlerMethodMap[method], name)
	methodMap := make(map[string]HandlerFunc)
	methodMap[method] = handlerFunc
	r.treeNode.Put(name)
}

type Engine struct {
	*router
}

func New() *Engine {
	return &Engine{
		router: &router{},
	}
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	for _, group := range e.router.groups {
		routerName := utils.SubStringLast(r.RequestURI, utils.ConcatenatedString([]string{"/", group.groupName}))
		node := group.treeNode.Get(routerName)
		if node != nil && node.isEnd { // 路由匹配成功
			ctx := &Context{
				W: w,
				R: r,
			}
			// ANY下的匹配
			handler, ok := group.handlerMap[node.routerName][MethodAny]
			if ok {
				handler(ctx)
				return
			}
			// 指定Method的匹配（如Get，Post）
			handler, ok = group.handlerMap[node.routerName][method]
			if ok {
				handler(ctx)
				return
			}
			// url匹配但请求方式不匹配，405 MethodNotAllowed
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintln(w, utils.ConcatenatedString([]string{method, " not allowed"}))
			return
		} else { // 路由匹配失败，404 NotFound
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, r.RequestURI+" not found")
		}
	}
}

func (e *Engine) Run() {
	http.Handle("/", e)
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		log.Fatalln(err)
	}
}
