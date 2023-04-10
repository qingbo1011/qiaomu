package qiaomu

import (
	"github.com/qingbo1011/qiaomu/render"
	"github.com/qingbo1011/qiaomu/utils"
	"html/template"
	"net/http"
	"net/url"
)

type Context struct {
	W          http.ResponseWriter
	R          *http.Request
	StatusCode int
	engine     *Engine
}

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
