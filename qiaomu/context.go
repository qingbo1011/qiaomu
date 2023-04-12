package qiaomu

import (
	"errors"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/qingbo1011/qiaomu/bind"
	"github.com/qingbo1011/qiaomu/render"
	"github.com/qingbo1011/qiaomu/utils"
)

const defaultMultipartMemory = 32 << 20 // 32M (ParseMultipartForm方法能支持的最大内存)

type Context struct {
	W                     http.ResponseWriter
	R                     *http.Request
	StatusCode            int
	engine                *Engine
	queryCache            url.Values
	formCache             url.Values
	DisallowUnknownFields bool
	IsValidate            bool
}

// Render 渲染统一处理
func (c *Context) Render(statusCode int, r render.Render) error {
	// 如果设置了statusCode，对header的修改就不生效了
	err := r.Render(c.W, statusCode)
	c.StatusCode = statusCode
	// 多次调用WriteHeader会产生这样的警告 superfluous response.WriteHeader
	return err
}

// HTML HTML页面渲染
func (c *Context) HTML(status int, html string) error {
	return c.Render(status, &render.HTML{Data: html, IsTemplate: false})
}

// HTMLTemplate HTML页面渲染：模板支持
func (c *Context) HTMLTemplate(name string, data any, filenames ...string) error {
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := template.New(name)
	t, err := t.ParseFiles(filenames...)
	if err != nil {
		return err
	}
	err = t.Execute(c.W, data)
	return err
}

// HTMLTemplateGlob 通过go html/template包自带的ParseGlob方法，实现filename的匹配模式
func (c *Context) HTMLTemplateGlob(name string, data any, pattern string) error {
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := template.New(name)
	t, err := t.ParseGlob(pattern)
	if err != nil {
		return err
	}
	err = t.Execute(c.W, data)
	return err
}

// Template 加载模板
func (c *Context) Template(name string, data any) error {
	return c.Render(http.StatusOK, &render.HTML{
		Data:       data,
		IsTemplate: true,
		Template:   c.engine.HTMLRender.Template,
		Name:       name,
	})
}

// JSON 渲染JSON数据
func (c *Context) JSON(status int, data any) error {
	return c.Render(status, &render.JSON{Data: data})
}

// XML 渲染XML数据
func (c *Context) XML(status int, data any) error {
	return c.Render(status, &render.XML{Data: data})
}

// File 文件下载支持
func (c *Context) File(fileName string) {
	http.ServeFile(c.W, c.R, fileName)
}

// FileAttachment 文件下载支持（可自定义下载后文件名称）
func (c *Context) FileAttachment(filepath, filename string) {
	if utils.IsASCII(filename) {
		c.W.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	} else {
		c.W.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.QueryEscape(filename))
	}
	http.ServeFile(c.W, c.R, filepath)
}

// FileFromFS 指定下载路径(filepath是相对文件系统的路径)
func (c *Context) FileFromFS(filepath string, fs http.FileSystem) {
	defer func(old string) {
		c.R.URL.Path = old
	}(c.R.URL.Path)
	c.R.URL.Path = filepath
	http.FileServer(fs).ServeHTTP(c.W, c.R)
}

// Redirect 重定向
func (c *Context) Redirect(status int, url string) error {
	return c.Render(status, &render.Redirect{
		Code:     status,
		Request:  c.R,
		Location: url,
	})
}

// String 渲染String字符串
func (c *Context) String(status int, format string, values ...any) error {
	return c.Render(status, &render.String{Format: format, Data: values})
}

// GetQuery 获取query参数 (以/user/add?name=李四 为例，可以获取到name参数为李四)
func (c *Context) GetQuery(key string) string {
	c.initQueryCache()
	return c.queryCache.Get(key)
}

// GetDefaultQuery 获取query参数，效果跟GetQuery类似，但是支持用户设置query参数的默认值，如果参数获取失败即提供用户预设值的默认值
func (c *Context) GetDefaultQuery(key, defaultValue string) string {
	values, ok := c.GetQueryArray(key)
	if !ok {
		return defaultValue
	}
	return values[0]
}

// GetQueryArray 获取query参数：将获取到的参数存储到string切片中(应有到同一个参数传多个的应用场景，如：/user/adds?id=111&id=222)
func (c *Context) GetQueryArray(key string) ([]string, bool) {
	c.initQueryCache()
	values, ok := c.queryCache[key]
	return values, ok
}

// QueryArray 效果跟GetQueryArray类似，只不过只返回切片，若切片为空即说明获取失败(false)
func (c *Context) QueryArray(key string) (values []string) {
	c.initQueryCache()
	values, _ = c.queryCache[key]
	return
}

// 初始化queryCache
func (c *Context) initQueryCache() {
	if c.R != nil {
		c.queryCache = c.R.URL.Query()
	} else {
		c.queryCache = url.Values{}
	}
}

// GetQueryMap map参数处理
func (c *Context) GetQueryMap(key string) (map[string]string, bool) {
	c.initQueryCache()
	return c.get(c.queryCache, key)
}

// QueryMap map参数处理(类似GetQueryMap，若处理失败返回空map)
func (c *Context) QueryMap(key string) map[string]string {
	dicts, _ := c.GetQueryMap(key)
	return dicts
}

// 获取url中的map参数(eg:/queryMap?user[id]=1&user[name]=张三)
func (c *Context) get(cache map[string][]string, key string) (map[string]string, bool) {
	dicts := make(map[string]string)
	exist := false
	// user[id]=1&user[name]=张三
	for k, value := range cache {
		if i := strings.IndexByte(k, '['); i >= 1 && k[0:i] == key {
			if j := strings.IndexByte(k[i+1:], ']'); j >= 1 {
				exist = true
				dicts[k[i+1:][:j]] = value[0]
			}
		}
	}
	return dicts, exist
}

// 初始化formCache
func (c *Context) initPostFormCache() {
	if c.R != nil {
		if err := c.R.ParseMultipartForm(defaultMultipartMemory); err != nil {
			if !errors.Is(err, http.ErrNotMultipart) {
				log.Println(err)
			}
		}
		c.formCache = c.R.PostForm
	} else {
		c.formCache = url.Values{}
	}
}

// GetPostFormArray 处理表单参数，返回结果为string切片
func (c *Context) GetPostFormArray(key string) ([]string, bool) {
	c.initPostFormCache()
	values, ok := c.formCache[key]
	return values, ok
}

// PostFormArray 处理表单参数(获取失败返回空切片)
func (c *Context) PostFormArray(key string) []string {
	values, _ := c.GetPostFormArray(key)
	return values
}

// GetPostForm 处理表单参数(返回string类型，取结果切片的第一个值)
func (c *Context) GetPostForm(key string) (string, bool) {
	if values, ok := c.GetPostFormArray(key); ok {
		return values[0], ok
	}
	return "", false
}

// GetPostFormMap 处理表单参数(将结果处理成map类型)
func (c *Context) GetPostFormMap(key string) (map[string]string, bool) {
	c.initPostFormCache()
	return c.get(c.formCache, key)
}

// PostFormMap 处理表单参数(将结果处理成map类型，处理失败返回空map)
func (c *Context) PostFormMap(key string) map[string]string {
	dicts, _ := c.GetPostFormMap(key)
	return dicts
}

// FormFile 表单参数处理中支持对文件参数的处理
func (c *Context) FormFile(name string) *multipart.FileHeader {
	file, header, err := c.R.FormFile(name)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()
	return header
}

// FormFiles 文件参数处理，支持多文件
func (c *Context) FormFiles(name string) []*multipart.FileHeader {
	multipartForm, err := c.MultipartForm()
	if err != nil {
		return make([]*multipart.FileHeader, 0)
	}
	return multipartForm.File[name]
}

// SaveUploadedFile 文件参数处理，支持用户自定义文件要上传的目录
func (c *Context) SaveUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

// MultipartForm 将form表单类型参数整个提取为*multipart.Form类型
func (c *Context) MultipartForm() (*multipart.Form, error) {
	err := c.R.ParseMultipartForm(defaultMultipartMemory)
	return c.R.MultipartForm, err
}

// BindJson 指定处理JSON参数
func (c *Context) BindJson(obj any) error {
	json := bind.JSON
	json.DisallowUnknownFields = true
	json.IsValidate = true
	return c.MustBindWith(obj, json)
}

// BindXML 指定处理XML参数
func (c *Context) BindXML(obj any) error {
	return c.MustBindWith(obj, bind.XML)
}

// MustBindWith 如果绑定出现错误，终止请求并返回400状态码
func (c *Context) MustBindWith(obj any, bind bind.Binding) error {
	if err := c.ShouldBind(obj, bind); err != nil {
		c.W.WriteHeader(http.StatusBadRequest)
		return err
	}
	return nil
}

// ShouldBind 如果绑定出现错误，返回错误并由开发者自行处理错误和请求
func (c *Context) ShouldBind(obj any, bind bind.Binding) error {
	return bind.Bind(c.R, obj)
}
