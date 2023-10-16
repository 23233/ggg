package pipe

import (
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"strings"
)

var (
	// HashGen hash生成 必传params Type可选b62 b58
	HashGen = &RunnerContext[any, *HashGenPipe, any, string]{
		Name: "hash生成",
		Key:  "hash_gen",
		call: func(ctx iris.Context, origin any, params *HashGenPipe, db any, more ...any) *RunResp[string] {
			if len(params.Cols) < 1 {
				return newPipeErr[string](PipePackParamsError)
			}

			var result string

			switch params.Type {
			case "b62":
				result = ut.StrToB62(strings.Join(params.Cols, ""))
				break
			case "b58":
				result = ut.StrToB58(strings.Join(params.Cols, ""))
				break
			default:
				return newPipeErr[string](errors.New("hash生成类型错误"))
			}
			return newPipeResult[string](result)
		},
	}
)

type HashGenPipe struct {
	Type string   `json:"type,omitempty"` // b58 b62
	Cols []string `json:"cols,omitempty"` // 字段 顺序执行
}
