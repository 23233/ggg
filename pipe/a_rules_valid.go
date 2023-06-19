package pipe

import (
	"github.com/23233/ggg/ut"
	"github.com/gookit/validate"
	"github.com/gookit/validate/locales/zhcn"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
)

// RuleValidate 使用这个库 https://github.com/gookit/validate/blob/master/README.zh-CN.md
// 格式是这样的 required|int|min:1|max:99
// 处理方法为
type RuleValidate struct {
	Key      string `json:"key,omitempty"`
	Value    any    `json:"value,omitempty"`
	Rules    string `json:"rules,omitempty"`     // 规则 看上面
	NeedType string `json:"need_type,omitempty"` // 类型转换 可为空
}

type RulesValidate struct {
	Record []*RuleValidate `json:"record,omitempty"`
}

type ValidResult struct {
	Pass bool   `json:"pass,omitempty"`
	Msg  string `json:"msg,omitempty"`
}

func (c *RulesValidate) Valid(data map[string]any) ValidResult {

	valid := validate.Map(data)
	// 注册中文
	zhcn.Register(valid)

	// 再挨个增加规则
	for _, rv := range c.Record {
		valid.StringRule(rv.Key, rv.Rules)
	}
	// 进行验证
	var result ValidResult
	result.Msg = "成功"
	pass := valid.Validate()
	if !pass {
		result.Msg = valid.Errors.One()
	}
	result.Pass = pass
	return result
}

var (
	RulesValid = &RunnerContext[any, *RulesValidate, any, ValidResult]{
		Name: "规则验证",
		Key:  "rules_valid",
		call: func(ctx iris.Context, origin any, params *RulesValidate, db any, more ...any) *PipeRunResp[ValidResult] {
			if params == nil {
				return newPipeErr[ValidResult](PipePackParamsError)
			}
			// 先获取到所有数据map
			mp := make(map[string]any)
			for _, rv := range params.Record {
				tv, err := ut.TypeChange(rv.Value, rv.NeedType)
				if err != nil {
					return newPipeErr[ValidResult](err)
				}
				mp[rv.Key] = tv
			}

			result := params.Valid(mp)
			if !result.Pass {
				return newPipeErr[ValidResult](errors.New(result.Msg))
			}

			return newPipeResult(result)

		},
	}
)
