package qiaomu

import (
	"fmt"
	"github.com/qingbo1011/qiaomu/utils"
	"log"
	"net/http"
)

type HandleFunc func(ctx *Context)

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
		handlerMap:       make(map[string]HandleFunc),
		handlerMethodMap: make(map[string][]string),
	}
	r.groups = append(r.groups, g)
	return g
}

type routerGroup struct {
	groupName        string
	handlerMap       map[string]HandleFunc
	handlerMethodMap map[string][]string
}

func (r *routerGroup) Any(name string, handleFunc HandleFunc) {
	r.handlerMap[name] = handleFunc
	r.handlerMethodMap["ANY"] = append(r.handlerMethodMap["ANY"], name)
}

func (r *routerGroup) Get(name string, handleFunc HandleFunc) {
	r.handlerMap[name] = handleFunc
	r.handlerMethodMap[http.MethodGet] = append(r.handlerMethodMap[http.MethodGet], name)
}

func (r *routerGroup) Post(name string, handleFunc HandleFunc) {
	r.handlerMap[name] = handleFunc
	r.handlerMethodMap[http.MethodPost] = append(r.handlerMethodMap[http.MethodPost], name)
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
		for name, handle := range group.handlerMap {
			url := utils.ConcatenatedString([]string{"/", group.groupName, name})
			if r.RequestURI == url {
				ctx := &Context{
					W: w,
					R: r,
				}
				// ANY下的匹配
				if group.handlerMethodMap["ANY"] != nil {
					for _, v := range group.handlerMethodMap["ANY"] {
						if name == v {
							handle(ctx)
							return
						}
					}
				}
				// 指定Method的匹配（如Get，Post）
				method := r.Method
				fmt.Println(method)
				routers := group.handlerMethodMap[method]
				if routers != nil {
					for _, v := range routers {
						if name == v {
							handle(ctx)
							return
						}
					}
				}
				w.WriteHeader(http.StatusMethodNotAllowed)
				fmt.Fprintln(w, utils.ConcatenatedString([]string{method, " not allowed"}))
				return
			}
		}
	}
}

func (e *Engine) Run() {
	//groups := e.router.groups
	//for _, group := range groups {
	//	for name, handle := range group.handlerMap {
	//		http.HandleFunc(utils.ConcatenatedString([]string{"/", group.groupName, name}), handle)
	//	}
	//}
	http.Handle("/", e)
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		log.Fatalln(err)
	}
}
