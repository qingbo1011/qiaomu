package qiaomu

import (
	"github.com/qingbo1011/qiaomu/render"
	"net/http"
)

type Context struct {
	W          http.ResponseWriter
	R          *http.Request
	StatusCode int
}

func (c *Context) Render(statusCode int, r render.Render) error {
	//如果设置了statusCode，对header的修改就不生效了
	err := r.Render(c.W, statusCode)
	c.StatusCode = statusCode
	//多次调用WriteHeader会产生这样的警告 superfluous response.WriteHeader
	return err
}

func (c *Context) HTML(status int, html string) error {
	return c.Render(status, &render.HTML{Data: html, IsTemplate: false})
}
