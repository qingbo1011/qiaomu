package qiaomu

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/qingbo1011/qiaomu/config"
	"github.com/qingbo1011/qiaomu/gateway"
	qlog "github.com/qingbo1011/qiaomu/log"
	"github.com/qingbo1011/qiaomu/register"
	"github.com/qingbo1011/qiaomu/render"
	"github.com/qingbo1011/qiaomu/utils"
)

const (
	MethodAny = "ANY"
)

type HandlerFunc func(ctx *Context)

type MiddlewareFunc func(handlerFunc HandlerFunc) HandlerFunc

type router struct {
	groups []*routerGroup
	engine *Engine
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
	g.Use(r.engine.middles...)
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

// 路由实现引入中间件
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

type ErrorHandler func(err error) (int, any)

type Engine struct {
	router
	funcMap          template.FuncMap
	HTMLRender       render.HTMLRender
	pool             sync.Pool
	Logger           *qlog.Logger
	middles          []MiddlewareFunc
	errorHandler     ErrorHandler
	OpenGateway      bool
	gatewayConfigs   []gateway.GWConfig
	gatewayTreeNode  *gateway.TreeNode
	gatewayConfigMap map[string]gateway.GWConfig
	RegisterType     string
	RegisterOption   register.Option
	RegisterCli      register.QueenRegister
}

func New() *Engine {
	engine := &Engine{
		router:           router{},
		gatewayTreeNode:  &gateway.TreeNode{Name: "/", Children: make([]*gateway.TreeNode, 0)},
		gatewayConfigMap: make(map[string]gateway.GWConfig),
	}
	engine.pool.New = func() any {
		return engine.allocateContext()
	}
	return engine
}

// Default 创建出来的engine自动注册日志中间件和错误处理中间件
func Default() *Engine {
	engine := New()
	engine.Logger = qlog.Default()
	logPath, ok := config.Conf.Log["path"]
	if ok {
		engine.Logger.SetLogPath(logPath.(string))
	}
	engine.Use(Logging, Recovery)
	engine.router.engine = engine
	return engine
}

func (e *Engine) Use(middles ...MiddlewareFunc) {
	e.middles = append(e.middles, middles...)
}

// 仿照gin框架源码作的处理
func (e *Engine) allocateContext() any {
	return &Context{engine: e}
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := e.pool.Get().(*Context)
	ctx.W = w
	ctx.R = r
	ctx.Logger = e.Logger
	e.httpRequestHandle(ctx, w, r)
	e.pool.Put(ctx)
}

func (e *Engine) httpRequestHandle(ctx *Context, w http.ResponseWriter, r *http.Request) {
	// 开启网关后的处理
	if e.OpenGateway {
		path := r.URL.Path
		node := e.gatewayTreeNode.Get(path)
		if node == nil {
			ctx.W.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(ctx.W, ctx.R.RequestURI+" not found")
			return
		}
		gwConfig := e.gatewayConfigMap[node.GwName]
		gwConfig.Header(ctx.R)
		addr, err := e.RegisterCli.GetValue(gwConfig.ServiceName)
		if err != nil {
			ctx.W.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(ctx.W, err.Error())
			return
		}
		target, err := url.Parse(fmt.Sprintf("http://%s%s", addr, path))
		if err != nil {
			ctx.W.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(ctx.W, err.Error())
			return
		}
		// 网关的处理逻辑
		director := func(req *http.Request) {
			req.Host = target.Host
			req.URL.Host = target.Host
			req.URL.Path = target.Path
			req.URL.Scheme = target.Scheme
			if _, ok := req.Header["User-Agent"]; !ok {
				req.Header.Set("User-Agent", "")
			}
		}
		response := func(response *http.Response) error {
			log.Println("响应修改")
			return nil
		}
		handler := func(writer http.ResponseWriter, request *http.Request, err error) {
			log.Println(err)
			log.Println("错误处理")
		}
		proxy := httputil.ReverseProxy{Director: director, ModifyResponse: response, ErrorHandler: handler}
		proxy.ServeHTTP(w, r)
		return
	}
	// 不开启网关的处理
	method := r.Method
	for _, group := range e.router.groups {
		routerName := utils.SubStringLast(r.URL.Path, utils.ConcatenatedString([]string{"/", group.groupName}))
		node := group.treeNode.Get(routerName)
		if node != nil && node.isEnd { // 路由匹配成功
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

func (e *Engine) SetFuncMap(funcMap template.FuncMap) {
	e.funcMap = funcMap
}

func (e *Engine) SetGatewayConfig(configs []gateway.GWConfig) {
	e.gatewayConfigs = configs
	// 把这个路径存储起来，在访问的时候去匹配这里面的路由，如果匹配，就设置相应的匹配结果
	for _, v := range e.gatewayConfigs {
		e.gatewayTreeNode.Put(v.Path, v.Name)
		e.gatewayConfigMap[v.Name] = v
	}
}

// LoadTemplate 加载模板
func (e *Engine) LoadTemplate(pattern string) {
	t := template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern))
	e.SetHtmlTemplate(t)
}

// LoadTemplateConf 根据配置文件读取模板
func (e *Engine) LoadTemplateConf() {
	pattern, ok := config.Conf.Template["pattern"]
	if ok {
		t := template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern.(string)))
		e.SetHtmlTemplate(t)
	}
}

// SetHtmlTemplate 加载HTML模板
func (e *Engine) SetHtmlTemplate(t *template.Template) {
	e.HTMLRender = render.HTMLRender{Template: t}
}

// RegisterErrorHandler 注册errorHandler
func (e *Engine) RegisterErrorHandler(handler ErrorHandler) {
	e.errorHandler = handler
}

func (e *Engine) Run(addr string) {
	if e.RegisterType == "nacos" {
		r := &register.QueenNacosRegister{}
		err := r.CreateCli(e.RegisterOption)
		if err != nil {
			panic(err)
		}
		e.RegisterCli = r
	}
	if e.RegisterType == "etcd" {
		r := &register.QueenEtcdRegister{}
		err := r.CreateCli(e.RegisterOption)
		if err != nil {
			panic(err)
		}
		e.RegisterCli = r
	}

	http.Handle("/", e)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

// RunTLS 支持https
func (e *Engine) RunTLS(addr, certFile, keyFile string) {
	err := http.ListenAndServeTLS(addr, certFile, keyFile, e.Handler())
	if err != nil {
		log.Fatal(err)
	}
}

// Handler 返回Handler
func (e *Engine) Handler() http.Handler {
	return e
}
