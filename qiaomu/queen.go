package qiaomu

import (
	"github.com/qingbo1011/qiaomu/utils"
	"log"
	"net/http"
)

type HandleFunc func(w http.ResponseWriter, r *http.Request)

type router struct {
	groups []*routerGroup
}

func (r *router) Group(name string) *routerGroup {
	g := &routerGroup{
		groupName:     name,
		handleFuncMap: make(map[string]HandleFunc),
	}
	r.groups = append(r.groups, g)
	return g
}

type routerGroup struct {
	groupName     string
	handleFuncMap map[string]HandleFunc
}

func (r *routerGroup) Add(name string, handleFunc HandleFunc) {
	r.handleFuncMap[name] = handleFunc
}

type Engine struct {
	*router
}

func New() *Engine {
	return &Engine{
		router: &router{},
	}
}

func (e *Engine) Run() {
	groups := e.router.groups
	for _, group := range groups {
		for name, handle := range group.handleFuncMap {
			http.HandleFunc(utils.ConcatenatedString([]string{"/", group.groupName, name}), handle)
		}
	}
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		log.Fatalln(err)
	}
}
