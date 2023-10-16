package pipe

import (
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
)

type KvValidConfig struct {
	Origin any    `json:"origin,omitempty"` // 原始值
	Op     string `json:"op,omitempty"`     // 操作符
	Value  string `json:"value,omitempty"`  // 值
}

func (c *KvValidConfig) Check() (bool, error) {
	var pass = false
	if len(c.Op) > 0 {
		pass = OpValid(c.Origin, c.Op, c.Value)
	} else {
		// 没有操作符直接对比值
		rawStr, err := ut.TypeChange(c.Origin, "string")
		if err != nil {
			return false, err
		}
		pass = rawStr.(string) == c.Value
	}

	if !pass {
		return pass, nil
	}

	return pass, nil
}

var (
	// KvValid kv结构验证器 必传params
	KvValid = &RunnerContext[any, *KvValidConfig, any, bool]{
		Name: "kv验证器",
		Key:  "kv_valid",
		call: func(ctx iris.Context, origin any, params *KvValidConfig, db any, more ...any) *RunResp[bool] {
			if params == nil {
				return newPipeErr[bool](PipePackParamsError)
			}

			pass, err := params.Check()
			if err != nil {
				return newPipeErr[bool](err)
			}

			return newPipeResult(pass)
		},
	}
	// pipeKvListValid 多个kv结构验证器 必传params
	pipeKvListValid = &RunnerContext[any, *KvsValidPipe, any, []bool]{
		Name: "kv验证器",
		Key:  "kv_valid",
		call: func(ctx iris.Context, origin any, params *KvsValidPipe, db any, more ...any) *RunResp[[]bool] {

			if params == nil {
				return newPipeErr[[]bool](PipePackParamsError)
			}
			var result = make([]bool, 0, len(params.Records))

			for _, kv := range params.Records {
				pass, err := kv.Check()
				if err != nil {
					return newPipeErr[[]bool](err)
				}
				result = append(result, pass)
			}

			return newPipeResult(result)
		},
	}
)

type KvsValidPipe struct {
	Records []*KvValidConfig `json:"records,omitempty" bson:"records,omitempty"`
}
