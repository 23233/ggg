package pipe

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"strings"
)

type ReqResponse struct {
	Type     string         `json:"type,omitempty"`     // 返回值类型 可选 string json 默认json
	Msg      string         `json:"msg,omitempty"`      // string的时候 这个msg存在 则直接吐出
	Items    map[string]any `json:"items,omitempty"`    // 返回值来源
	Excludes []string       `json:"excludes,omitempty"` // 返回值排除 仅type为json时有效
}

type ParseResponse struct {
	Msg  string `json:"msg,omitempty"`
	Json any    `json:"json,omitempty"`
}

var (
	// ResponseParse 请求返回值的设定与解析
	// 必传params ReqResponse
	ResponseParse = &RunnerContext[any, *ReqResponse, any, *ParseResponse]{
		Name: "返回值解析",
		Key:  "response_parse",
		call: func(ctx iris.Context, origin any, params *ReqResponse, db any, more ...any) *RunResp[*ParseResponse] {

			if params == nil {
				return NewPipeErr[*ParseResponse](PipePackParamsError)
			}

			var pr = new(ParseResponse)
			// 如果只是返回string
			if params.Type == "string" && len(params.Msg) >= 1 {
				pr.Msg = params.Msg
			} else {
				pr.Json = nil

				if params.Items != nil {

					respMap := params.Items

					// 进行response 返回值排除
					if len(params.Excludes) > 0 {
						for _, exclude := range params.Excludes {
							tls := strings.Split(exclude, ".")
							if len(tls) == 1 {
								delete(respMap, exclude)
								continue
							}
							// 暂时只支持2级 value且仅为map
							if len(tls) == 2 {
								top := tls[0]
								if v, ok := respMap[top]; ok {
									if vt, ok := v.(map[string]any); ok {
										delete(vt, tls[1])
									}
								}
							}
						}
					}

					switch params.Type {
					case "string":
						st, err := jsoniter.MarshalToString(respMap)
						if err != nil {
							return NewPipeErr[*ParseResponse](errors.Wrap(err, "返回值转换为字符串失败"))
						}
						pr.Msg = st
						break
					default:
						pr.Json = respMap
						break
					}

				}

			}
			return NewPipeResult(pr)
		},
	}
)
