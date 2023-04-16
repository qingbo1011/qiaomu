package qiaomu

import (
	"encoding/base64"
	"net/http"

	"github.com/qingbo1011/qiaomu/utils"
)

type Accounts struct {
	UnAuthHandler func(ctx *Context)
	Users         map[string]string
	Realm         string
}

// BasicAuth Basic认证中间件
func (a *Accounts) BasicAuth(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		username, password, ok := ctx.R.BasicAuth()
		if !ok {
			a.unAuthHandler(ctx)
			return
		}
		pwd, exist := a.Users[username]
		if !exist {
			a.unAuthHandler(ctx)
			return
		}
		if pwd != password {
			a.unAuthHandler(ctx)
			return
		}
		ctx.Set("user", username)
		next(ctx)
	}
}

func (a *Accounts) unAuthHandler(ctx *Context) {
	if a.UnAuthHandler != nil {
		a.UnAuthHandler(ctx)
	} else {
		ctx.W.Header().Set("WWW-Authenticate", a.Realm)
		ctx.W.WriteHeader(http.StatusUnauthorized)
	}
}

// BasicAuth 根据username和password获取Basic认证的Base64字符串
func BasicAuth(username, password string) string {
	auth := utils.ConcatenatedString([]string{username, ":", password})
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
