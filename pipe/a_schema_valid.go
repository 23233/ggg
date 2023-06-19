package pipe

import (
	"github.com/23233/jsonschema"
	jsoniter "github.com/json-iterator/go"
	"github.com/kataras/iris/v12"
)

type SchemaValidConfig struct {
	Selector map[string]any     `json:"selector,omitempty"` // 选择器 选择 bucket中的那些字段 会组合成 map[string]any
	Schema   *jsonschema.Schema `json:"schema,omitempty"`   // 验证器 使用 schema 进行验证
}

var (
	SchemaValid = &RunnerContext[any, *SchemaValidConfig, any, map[string]any]{
		Name: "schema验证器",
		Key:  "schema_valid",
		call: func(ctx iris.Context, origin any, params *SchemaValidConfig, db any, more ...any) *RunResp[map[string]any] {
			// schema 验证
			if params == nil {
				return newPipeErr[map[string]any](PipePackParamsError)
			}

			rawBin, err := jsoniter.Marshal(params.Schema)
			if err != nil {
				return newPipeErr[map[string]any](err)
			}

			err = SchemaValidFunc(rawBin, params.Selector)
			if err != nil {
				return newPipeErr[map[string]any](err)
			}

			return newPipeResult(params.Selector)
		},
	}
)
