package pipe

import (
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
)

var (
	// ModelMapper 模型body中映射取出对应的map 不需要db
	// 必传origin 支持map和struct struct以json的tag为key进行匹配
	// 必传params ut.ModelCtxMapperPack new(ut.ModelCtxMapperPack)都可以 但是必传
	ModelMapper = &RunnerContext[any, *ut.ModelCtxMapperPack, any, any]{
		Key:  "model_ctx_mapper",
		Name: "模型body取map",
		call: func(ctx iris.Context, origin any, params *ut.ModelCtxMapperPack, db any, more ...any) *RunResp[any] {
			var bodyData = origin
			if origin == nil {
				return NewPipeErr[any](PipeOriginError)
			}
			err := params.Process(bodyData)
			if err != nil {
				return NewPipeErr[any](err)
			}

			return NewPipeResult(bodyData)

		},
	}
)
