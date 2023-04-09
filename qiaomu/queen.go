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

type MiddlewareFunc func(handlerFunc HandlerFunc) HandlerFunc

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
		middlewaresFuncMap: make(map[string]map[string][]MiddlewareFunc),
	}
	r.groups = append(r.groups, g)
	return g
}

type routerGroup struct {
	groupName          string
	handlerMap         map[string]map[string]HandlerFunc
	handlerMethodMap   map[string][]string
	treeNode           *treeNode
	middlewaresFuncMap map[string]map[string][]MiddlewareFunc
	middlewares        []MiddlewareFunc
}

// Handle method的有效性校验
func (r *routerGroup) Handle(name string, method string, handlerFunc HandlerFunc) {
	r.handle(name, method, handlerFunc)
}

// Any 任意类型的路由
func (r *routerGroup) Any(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, MethodAny, handleFunc, middlewareFunc...)
}

// Get Get类型路由
func (r *routerGroup) Get(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodGet, handleFunc, middlewareFunc...)
}

// Head Head类型路由
func (r *routerGroup) Head(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodHead, handleFunc, middlewareFunc...)
}

// Post Post类型路由
func (r *routerGroup) Post(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPost, handleFunc, middlewareFunc...)
}

// Put Put类型路由
func (r *routerGroup) Put(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPut, handleFunc, middlewareFunc...)
}

// Patch Patch类型路由
func (r *routerGroup) Patch(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPatch, handleFunc, middlewareFunc...)
}

// Delete Delete类型路由
func (r *routerGroup) Delete(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodDelete, handleFunc, middlewareFunc...)
}

// Connect Connect类型路由
func (r *routerGroup) Connect(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodConnect, handleFunc, middlewareFunc...)
}

// Options Options类型路由
func (r *routerGroup) Options(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodOptions, handleFunc, middlewareFunc...)
}

// Trace Trace类型路由
func (r *routerGroup) Trace(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodTrace, handleFunc, middlewareFunc...)
}

// 统一处理
func (r *routerGroup) handle(name string, method string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	_, ok := r.handlerMap[name]
	if !ok {
		r.handlerMap[name] = make(map[string]HandlerFunc)
		r.middlewaresFuncMap[name] = make(map[string][]MiddlewareFunc)
	}
	r.handlerMap[name][method] = handlerFunc
	r.handlerMethodMap[method] = append(r.handlerMethodMap[method], name)
	methodMap := make(map[string]HandlerFunc)
	methodMap[method] = handlerFunc
	r.middlewaresFuncMap[name][method] = append(r.middlewaresFuncMap[name][method], middlewareFunc...) // 添加中间件
	r.treeNode.Put(name)
}

func (r *routerGroup) methodHandle(ctx *Context, name string, method string, handler HandlerFunc) {
	// 路由组级中间件
	if r.middlewares != nil {
		for _, middlewareFunc := range r.middlewares {
			handler = middlewareFunc(handler)
		}
	}
	// 路由级中间件
	middlewareFuncs := r.middlewaresFuncMap[name][method]
	if middlewareFuncs != nil {
		for _, middlewareFunc := range middlewareFuncs {
			handler = middlewareFunc(handler)
		}
	}
	// 执行路由逻辑
	handler(ctx)
}

// Use 注册中间件
func (r *routerGroup) Use(middlewareFunc ...MiddlewareFunc) {
	r.middlewares = append(r.middlewares, middlewareFunc...)
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
	e.httpRequestHandle(w, r)
}

func (e *Engine) httpRequestHandle(w http.ResponseWriter, r *http.Request) {
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
				group.methodHandle(ctx, node.routerName, MethodAny, handler)
				return
			}
			// 指定Method的匹配（如Get，Post）
			handler, ok = group.handlerMap[node.routerName][method]
			if ok {
				group.methodHandle(ctx, node.routerName, method, handler)
				return
			}
			// url匹配但请求方式不匹配，405 MethodNotAllowed
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintln(w, utils.ConcatenatedString([]string{method, " not allowed"}))
			return
		}
	}
	// 路由匹配失败，404 NotFound
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintln(w, r.RequestURI+" not found")
}

func (e *Engine) Run() {
	http.Handle("/", e)
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		log.Fatalln(err)
	}
}
