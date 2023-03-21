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
	}
	r.groups = append(r.groups, g)
	return g
}

type routerGroup struct {
	groupName        string
	handlerMap       map[string]map[string]HandlerFunc
	handlerMethodMap map[string][]string
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
	groups := e.router.groups
	for _, group := range groups {
		for name, methodHandle := range group.handlerMap {
			url := utils.ConcatenatedString([]string{"/", group.groupName, name})
			if r.RequestURI == url { // url匹配
				ctx := &Context{
					W: w,
					R: r,
				}
				// ANY下的匹配
				handler, ok := methodHandle[MethodAny]
				if ok {
					handler(ctx)
					return
				}
				// 指定Method的匹配（如Get，Post）
				method := r.Method
				handler, ok = methodHandle[method]
				if ok {
					handler(ctx)
					return
				}
				// url匹配但请求方式不匹配，405 MethodNotAllowed
				w.WriteHeader(http.StatusMethodNotAllowed)
				fmt.Fprintln(w, utils.ConcatenatedString([]string{method, " not allowed"}))
				return
			} else { // url不匹配，404 NotFound
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintln(w, r.RequestURI+" not found")
				return
			}
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
