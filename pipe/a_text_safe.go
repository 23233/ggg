package pipe

import (
	"fmt"
	"github.com/23233/ggg/contentSafe"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
)

var (
	// TextSafe 文字安全校验
	// 必传origin 为待校验文字
	TextSafe = &RunnerContext[string, any, any, bool]{
		Name: "文字安全验证",
		Key:  "text_safe",
		call: func(ctx iris.Context, origin string, params any, db any, more ...any) *RunResp[bool] {

			value := origin
			if len(value) > 1 {
				result, msg := contentSafe.C.AutoHitText(value)
				if !result {
					return NewPipeResultErr(result, errors.New(fmt.Sprintf("文字安全校验失败 错误:%s", msg)))
				}
				return NewPipeResult(result)
			}

			return NewPipeResult(false)
		},
	}
)
