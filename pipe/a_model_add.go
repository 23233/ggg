package pipe

import (
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/qiniu/qmgo"
	"reflect"
)

// 从body json中获取出新增的内容

type ModelCtxAddConfig struct {
	ModelId string `json:"model_id,omitempty"`
}

var (
	// ModelAdd 模型json新增
	// 必传origin 支持map和struct 为模型的struct 对于传入struct会转换为map 通过json标签为key
	// 必传params ModelCtxAddConfig 配置信息 模型ID必须存在
	// 必传db 为qmgo的Database的实例
	ModelAdd = &RunnerContext[any, *ModelCtxAddConfig, *qmgo.Database, map[string]any]{
		Key:  "model_ctx_add",
		Name: "模型json新增",
		call: func(ctx iris.Context, origin any, params *ModelCtxAddConfig, db *qmgo.Database, more ...any) *RunResp[map[string]any] {
			if origin == nil {
				return NewPipeErr[map[string]any](PipeDepNotFound)
			}

			rawData := make(map[string]any)

			typ := reflect.TypeOf(origin)
			if typ.Kind() == reflect.Pointer {
				typ = typ.Elem()
			}

			switch typ.Kind() {
			case reflect.Struct:
				mp, err := ut.StructToMap(origin)
				if err != nil {
					return NewPipeErr[map[string]any](err)
				}
				rawData = mp
			case reflect.Map:
				rawData = origin.(map[string]any)
			default:
				return NewPipeErr[map[string]any](errors.New("origin 类型错误"))
			}

			// 注入_id
			mp := DefaultModelMap()
			mapper := &ut.ModelCtxMapperPack{
				InjectData: mp,
			}
			err := mapper.Process(rawData)
			if err != nil {
				return NewPipeErr[map[string]any](err)
			}

			_, err = db.Collection(params.ModelId).InsertOne(ctx, rawData)
			if err != nil {
				return NewPipeErr[map[string]any](err)
			}

			return NewPipeResult(rawData)
		},
	}
)
