package pipe

import (
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
)

type RandomPipe struct {
	NeedType string  `json:"need_type,omitempty"` // 需要的类型 string int uint float
	Len      int64   `json:"len,omitempty"`       // 对string时有效 string 时必设
	Start    float64 `json:"start,omitempty"`     // 对数字类有效 起始数字
	End      float64 `json:"end,omitempty"`       // 对数字类有效 截止数字
}

var (
	// RandomGen 随机数生成
	// 必传params RandomPipe
	RandomGen = &RunnerContext[any, *RandomPipe, any, any]{
		Name: "随机数生成",
		Key:  "random",
		call: func(ctx iris.Context, origin any, params *RandomPipe, db any, more ...any) *RunResp[any] {

			var result any

			switch params.NeedType {
			case "string":
				if params.Len < 1 {
					return NewPipeErr[any](errors.New("需要指定生成的字符串位数"))
				}
				result = ut.RandomStr(int(params.Len))
				break
			case "int":
				if params.End < params.Start {
					return NewPipeErr[any](errors.New("截止值应该大于起始值"))

				}
				result = ut.RandomInt(int(params.End), int(params.Start))
				break
			default:
				return NewPipeErr[any](errors.New("生成器类型未被支持"))
			}

			return NewPipeResult(result)
		},
	}
)
