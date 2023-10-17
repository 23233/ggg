package pipe

import (
	"github.com/23233/ggg/ut"
	"os"
	"time"
)

// RunResp 操作序列执行结果
type RunResp[T any] struct {
	Result       T     // 执行结果
	Err          error // 错误
	ReqCode      int   // 请求状态码 权重在pipeline定义的errCode之后
	BusinessCode int   // 业务code 仅在错误时有则会返回
	IsBreak      bool  // 是否中断之后的执行
}

func (c *RunResp[T]) SetBusinessCode(businessCode int) *RunResp[T] {
	c.BusinessCode = businessCode
	return c
}
func (c *RunResp[T]) SetReqCode(reqCode int) *RunResp[T] {
	c.ReqCode = reqCode
	return c
}
func (c *RunResp[T]) SetBreak(b bool) *RunResp[T] {
	c.IsBreak = b
	return c
}

func newPipeErr[T any](err error) *RunResp[T] {
	return &RunResp[T]{
		Err: err,
	}
}
func newPipeResult[T any](result T) *RunResp[T] {
	return &RunResp[T]{
		Result: result,
	}
}
func newPipeResultErr[T any](result T, err error) *RunResp[T] {
	return &RunResp[T]{
		Err:    err,
		Result: result,
	}
}

type StrTemplate struct {
	VarName string `json:"var_name,omitempty"`
	Value   any    `json:"value,omitempty"`
}

// StrExpand 字符串展开 字符串模板
type StrExpand struct {
	Key    string        `json:"key,omitempty"`
	KeyMap []StrTemplate `json:"key_map,omitempty"`
}

func (c *StrExpand) Build() (string, error) {
	if c == nil {
		return "", nil
	}
	attach := c.Key
	if len(c.KeyMap) > 0 {
		am := make(map[string]string)
		// 注入常量
		am["now_rfc3339"] = time.Now().Format(time.RFC3339)
		am["now"] = time.Now().Format("2006-01-02 15:04:05")
		for _, temp := range c.KeyMap {
			v, err := ut.TypeChange(temp.Value, "string")
			if err != nil {
				return "", err
			}
			vv := v.(string)
			am[temp.VarName] = vv
		}
		attach = os.Expand(attach, func(k string) string {
			v, ok := am[k]
			if !ok {
				return ""
			}
			return v
		})
	}
	return attach, nil
}
