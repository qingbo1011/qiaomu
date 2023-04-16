package token

import (
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/qingbo1011/qiaomu"
)

const JWTToken = "qiaomu_token"

type JwtHandler struct {
	Alg            string        // 指定jwt的算法
	TimeOut        time.Duration // token过期时间
	RefreshTimeOut time.Duration
	TimeFuc        func() time.Time // 时间函数
	Key            []byte           // Key
	RefreshKey     string           // 刷新key
	PrivateKey     string           // 私钥
	SendCookie     bool
	Authenticator  func(ctx *qiaomu.Context) (map[string]any, error)
	CookieName     string
	CookieMaxAge   int64
	CookieDomain   string
	SecureCookie   bool
	CookieHTTPOnly bool
	Header         string
	AuthHandler    func(ctx *qiaomu.Context, err error)
}

type JwtResponse struct {
	Token        string
	RefreshToken string // 刷新token，比原token生效时间长。防止用户过期时间一到就要要重新登录(当用户登录token失效后而此时用户正在使用应用，则不用该将用户直接登出重新登录，拿RefreshToken替代原token)
}

// LoginHandler 登录后的jwt处理
func (j *JwtHandler) LoginHandler(ctx *qiaomu.Context) (*JwtResponse, error) {
	data, err := j.Authenticator(ctx)
	if err != nil {
		return nil, err
	}
	if j.Alg == "" {
		j.Alg = "HS256"
	}
	// A部分
	signingMethod := jwt.GetSigningMethod(j.Alg)
	token := jwt.New(signingMethod)
	//  B部分
	claims := token.Claims.(jwt.MapClaims)
	if data != nil {
		for key, value := range data {
			claims[key] = value
		}
	}
	if j.TimeFuc == nil {
		j.TimeFuc = func() time.Time {
			return time.Now()
		}
	}
	expire := j.TimeFuc().Add(j.TimeOut)
	claims["exp"] = expire.Unix() // 设置过期时间
	claims["iat"] = j.TimeFuc().Unix()
	var tokenString string
	var tokenErr error
	//C部分 secret
	if j.usingPublicKeyAlgo() {
		tokenString, tokenErr = token.SignedString(j.PrivateKey)
	} else {
		tokenString, tokenErr = token.SignedString(j.Key)
	}
	if tokenErr != nil {
		return nil, tokenErr
	}
	jr := &JwtResponse{
		Token: tokenString,
	}
	// refreshToken
	refreshToken, err := j.refreshToken(token)
	if err != nil {
		return nil, err
	}
	jr.RefreshToken = refreshToken
	//  发送存储cookie
	if j.SendCookie {
		if j.CookieName == "" {
			j.CookieName = JWTToken
		}
		if j.CookieMaxAge == 0 {
			j.CookieMaxAge = expire.Unix() - j.TimeFuc().Unix()
		}
		ctx.SetCookie(j.CookieName, tokenString, int(j.CookieMaxAge), "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
	}
	return jr, nil
}

func (j *JwtHandler) usingPublicKeyAlgo() bool {
	switch j.Alg {
	case "RS256", "RS512", "RS384":
		return true
	}
	return false
}

func (j *JwtHandler) refreshToken(token *jwt.Token) (string, error) {
	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = j.TimeFuc().Add(j.RefreshTimeOut).Unix()
	var tokenString string
	var tokenErr error
	if j.usingPublicKeyAlgo() {
		tokenString, tokenErr = token.SignedString(j.PrivateKey)
	} else {
		tokenString, tokenErr = token.SignedString(j.Key)
	}
	if tokenErr != nil {
		return "", tokenErr
	}
	return tokenString, nil
}

// LogoutHandler 退出登录，旧的token就不再生效了(无论是否过期)
func (j *JwtHandler) LogoutHandler(ctx *qiaomu.Context) error {
	if j.SendCookie {
		if j.CookieName == "" {
			j.CookieName = JWTToken
		}
		ctx.SetCookie(j.CookieName, "", -1, "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
		return nil
	}
	return nil
}

// RefreshHandler 刷新token
func (j *JwtHandler) RefreshHandler(ctx *qiaomu.Context) (*JwtResponse, error) {
	rToken, ok := ctx.Get(j.RefreshKey)
	if !ok {
		return nil, errors.New("refresh token is null")
	}
	if j.Alg == "" {
		j.Alg = "HS256"
	}
	// 解析token
	t, err := jwt.Parse(rToken.(string), func(token *jwt.Token) (interface{}, error) {
		if j.usingPublicKeyAlgo() {
			return j.PrivateKey, nil
		} else {
			return j.Key, nil
		}
	})
	if err != nil {
		return nil, err
	}
	//  B部分
	claims := t.Claims.(jwt.MapClaims)

	if j.TimeFuc == nil {
		j.TimeFuc = func() time.Time {
			return time.Now()
		}
	}
	expire := j.TimeFuc().Add(j.TimeOut)
	claims["exp"] = expire.Unix() // 设置过期时间
	claims["iat"] = j.TimeFuc().Unix()
	var tokenString string
	var tokenErr error
	// C部分 secret
	if j.usingPublicKeyAlgo() {
		tokenString, tokenErr = t.SignedString(j.PrivateKey)
	} else {
		tokenString, tokenErr = t.SignedString(j.Key)
	}
	if tokenErr != nil {
		return nil, tokenErr
	}
	jr := &JwtResponse{
		Token: tokenString,
	}
	refreshToken, err := j.refreshToken(t)
	if err != nil {
		return nil, err
	}
	jr.RefreshToken = refreshToken
	// 发送存储cookie
	if j.SendCookie {
		if j.CookieName == "" {
			j.CookieName = JWTToken
		}
		if j.CookieMaxAge == 0 {
			j.CookieMaxAge = expire.Unix() - j.TimeFuc().Unix()
		}
		ctx.SetCookie(j.CookieName, tokenString, int(j.CookieMaxAge), "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
	}
	return jr, nil
}

// AuthInterceptor jwt登录中间件
func (j *JwtHandler) AuthInterceptor(next qiaomu.HandlerFunc) qiaomu.HandlerFunc {
	return func(ctx *qiaomu.Context) {
		if j.Header == "" {
			j.Header = "Authorization"
		}
		token := ctx.R.Header.Get(j.Header)
		if token == "" {
			if j.SendCookie {
				cookie, err := ctx.R.Cookie(j.CookieName)
				if err != nil {
					j.AuthErrorHandler(ctx, err)
					return
				}
				token = cookie.String()
			}
		}
		if token == "" {
			j.AuthErrorHandler(ctx, errors.New("token is null"))
			return
		}

		//解析token
		t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			if j.usingPublicKeyAlgo() {
				return j.PrivateKey, nil
			} else {
				return j.Key, nil
			}
		})
		if err != nil {
			j.AuthErrorHandler(ctx, err)
			return
		}
		claims := t.Claims.(jwt.MapClaims)
		ctx.Set("jwt_claims", claims)
		next(ctx)
	}
}

// AuthErrorHandler 认证错误处理
func (j *JwtHandler) AuthErrorHandler(ctx *qiaomu.Context, err error) {
	if j.AuthHandler == nil {
		ctx.W.WriteHeader(http.StatusUnauthorized)
	} else {
		j.AuthHandler(ctx, err)
	}
}
