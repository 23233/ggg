package pipe

import (
	"fmt"
	"github.com/imroc/req/v3"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"net/http"
	"strings"
)

type HttpRequestConfig struct {
	Uri        string            `json:"uri,omitempty"`         // 请求地址
	Method     string            `json:"method,omitempty"`      // 请求方法
	PathParams map[string]string `json:"path_params,omitempty"` //
	UrlParams  map[string]any    `json:"url_params,omitempty"`  //
	Body       any               `json:"body,omitempty"`        //
	Headers    map[string]string `json:"headers,omitempty"`     // 请求头设置
	MustCode   int               `json:"must_code,omitempty"`   // 响应码
}

func (c *HttpRequestConfig) GetMethod() string {
	if len(c.Method) < 1 {
		return http.MethodGet
	}
	return strings.ToUpper(c.Method)
}

func (c *HttpRequestConfig) GetMustCode() int {
	if c.MustCode < 1 {
		return 200
	}
	return c.MustCode
}

var (
	HttpRequest = &RunnerContext[any, *HttpRequestConfig, any, *req.Response]{
		Name: "http请求",
		Key:  "http_request",
		call: func(ctx iris.Context, origin any, params *HttpRequestConfig, db any, more ...any) *PipeRunResp[*req.Response] {
			if params == nil {
				return newPipeErr[*req.Response](PipePackParamsError)
			}
			// 格式验证
			if len(params.Uri) < 1 {
				return newPipeErr[*req.Response](PipeParamsError)
			}

			// 设置请求头
			r := req.R()
			r.SetHeaders(params.Headers)

			// 设置路径参数
			if params.PathParams != nil {
				r.SetPathParams(params.PathParams)
			}
			// 设置url query
			if params.UrlParams != nil {
				r.SetQueryParamsAnyType(params.UrlParams)
			}

			// 如果不是get 则可以设置body
			if params.GetMethod() != http.MethodGet {
				if params.Body != nil {
					r.SetBody(params.Body)
				}
			}

			resp, err := r.Send(params.GetMethod(), params.Uri)
			if err != nil {
				return newPipeErr[*req.Response](err)
			}
			if resp.StatusCode != params.GetMustCode() {
				return newPipeErr[*req.Response](errors.New(fmt.Sprintf("请求需要 %d 响应码 但得到了 %d", params.GetMustCode(), resp.StatusCode)))
			}
			if resp.IsErrorState() {
				return newPipeErr[*req.Response](resp.Err)
			}

			return newPipeResult[*req.Response](resp)
		},
	}
)
