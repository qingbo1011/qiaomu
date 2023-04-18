package rpc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	qlog "github.com/qingbo1011/qiaomu/log"
)

type QueenHttpClient struct {
	client     http.Client
	serviceMap map[string]QueenService
}

func NewHttpClient() *QueenHttpClient {
	// Transport 请求分发  协程安全 连接池
	client := http.Client{
		Timeout: time.Duration(3) * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   5,
			MaxConnsPerHost:       100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	return &QueenHttpClient{client: client, serviceMap: make(map[string]QueenService)}
}

func (c *QueenHttpClient) GetRequest(method string, url string, args map[string]any) (*http.Request, error) {
	if args != nil && len(args) > 0 {
		url = url + "?" + c.toValues(args)
	}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (c *QueenHttpClient) FormRequest(method string, url string, args map[string]any) (*http.Request, error) {
	req, err := http.NewRequest(method, url, strings.NewReader(c.toValues(args)))
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (c *QueenHttpClient) JsonRequest(method string, url string, args map[string]any) (*http.Request, error) {
	jsonStr, _ := json.Marshal(args)
	req, err := http.NewRequest(method, url, bytes.NewReader(jsonStr))
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (c *QueenHttpClientSession) Response(req *http.Request) ([]byte, error) {
	return c.responseHandle(req)
}

func (c *QueenHttpClientSession) Get(url string, args map[string]any) ([]byte, error) {
	// get请求的参数
	if args != nil && len(args) > 0 {
		url = url + "?" + c.toValues(args)
	}
	logger := qlog.Default()
	logger.Info(url)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return c.responseHandle(request)
}

func (c *QueenHttpClientSession) PostForm(url string, args map[string]any) ([]byte, error) {
	request, err := http.NewRequest("POST", url, strings.NewReader(c.toValues(args)))
	if err != nil {
		return nil, err
	}
	return c.responseHandle(request)
}

func (c *QueenHttpClientSession) PostJson(url string, args map[string]any) ([]byte, error) {
	marshal, _ := json.Marshal(args)
	request, err := http.NewRequest("POST", url, bytes.NewReader(marshal))
	if err != nil {
		return nil, err
	}
	return c.responseHandle(request)
}

func (c *QueenHttpClientSession) responseHandle(request *http.Request) ([]byte, error) {
	c.ReqHandler(request)
	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		info := fmt.Sprintf("response status is %d", response.StatusCode)
		return nil, errors.New(info)
	}
	reader := bufio.NewReader(response.Body)

	defer response.Body.Close()

	var buf = make([]byte, 127)
	var body []byte
	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if err == io.EOF || n == 0 {
			break
		}
		body = append(body, buf[:n]...)
		if n < len(buf) {
			break
		}
	}
	return body, nil

}

func (c *QueenHttpClient) toValues(args map[string]any) string {
	if args != nil && len(args) > 0 {
		params := url.Values{}
		for k, v := range args {
			params.Set(k, fmt.Sprintf("%v", v))
		}
		return params.Encode()
	}
	return ""
}

type HttpConfig struct {
	Protocol string
	Host     string
	Port     int
}

const (
	HTTP  = "http"
	HTTPS = "https"
)
const (
	GET      = "GET"
	POSTForm = "POST_FORM"
	POSTJson = "POST_JSON"
)

type QueenService interface {
	Env() HttpConfig
}

type QueenHttpClientSession struct {
	*QueenHttpClient
	ReqHandler func(req *http.Request)
}

func (c *QueenHttpClient) RegisterHttpService(name string, service QueenService) {
	c.serviceMap[name] = service
}

func (c *QueenHttpClient) Session() *QueenHttpClientSession {
	return &QueenHttpClientSession{
		c, nil,
	}
}

func (c *QueenHttpClientSession) Do(service string, method string) QueenService {
	queenService, ok := c.serviceMap[service]
	if !ok {
		panic(errors.New("service not found"))
	}
	// 找到service里面的Field,给其中要调用的方法赋值
	t := reflect.TypeOf(queenService)
	v := reflect.ValueOf(queenService)
	if t.Kind() != reflect.Pointer {
		panic(errors.New("service not pointer"))
	}
	tVar := t.Elem()
	vVar := v.Elem()
	fieldIndex := -1
	for i := 0; i < tVar.NumField(); i++ {
		name := tVar.Field(i).Name
		if name == method {
			fieldIndex = i
			break
		}
	}
	if fieldIndex == -1 {
		panic(errors.New("method not found"))
	}
	tag := tVar.Field(fieldIndex).Tag
	rpcInfo := tag.Get("qrpc")
	if rpcInfo == "" {
		panic(errors.New("not qrpc tag"))
	}
	split := strings.Split(rpcInfo, ",")
	if len(split) != 2 {
		panic(errors.New("tag qrpc not valid"))
	}
	methodType := split[0]
	path := split[1]
	httpConfig := queenService.Env()

	f := func(args map[string]any) ([]byte, error) {
		if methodType == GET {
			return c.Get(httpConfig.Prefix()+path, args)
		}
		if methodType == POSTForm {
			return c.PostForm(httpConfig.Prefix()+path, args)
		}
		if methodType == POSTJson {
			return c.PostJson(httpConfig.Prefix()+path, args)
		}
		return nil, errors.New("no match method type")
	}
	fValue := reflect.ValueOf(f)
	vVar.Field(fieldIndex).Set(fValue)
	return queenService
}

func (c HttpConfig) Prefix() string {
	if c.Protocol == "" {
		c.Protocol = HTTP
	}
	switch c.Protocol {
	case HTTP:
		return fmt.Sprintf("http://%s:%d", c.Host, c.Port)
	case HTTPS:
		return fmt.Sprintf("https://%s:%d", c.Host, c.Port)
	}
	return ""
}
