package pipe

import (
	"github.com/gookit/goutil/structs"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"time"
)

// 从body json中获取出新增的内容

type ModelCtxAddConfig struct {
	ModelId string `json:"model_id,omitempty"`
}

func DefaultModelMap() map[string]any {
	var m = make(map[string]any, 0)
	m["_id"] = primitive.NewObjectID()
	m["uid"] = SfNextId()
	m["update_at"] = time.Now().Local()
	m["create_at"] = time.Now().Local()
	return m
}

var (
	// ModelAdd 模型新增origin支持struct和map的传入 对于传入struct会转换为map 通过json标签为key
	ModelAdd = &RunnerContext[any, *ModelCtxAddConfig, *qmgo.Database, map[string]any]{
		Key:  "model_ctx_add",
		Name: "模型json新增",
		call: func(ctx iris.Context, origin any, params *ModelCtxAddConfig, db *qmgo.Database, more ...any) *RunResp[map[string]any] {
			if origin == nil {
				return newPipeErr[map[string]any](PipeDepNotFound)
			}

			rawData := make(map[string]any)

			typ := reflect.TypeOf(origin)
			if typ.Kind() == reflect.Pointer {
				typ = typ.Elem()
			}

			switch typ.Kind() {
			case reflect.Struct:
				mp, err := structs.StructToMap(origin)
				if err != nil {
					return newPipeErr[map[string]any](err)
				}
				rawData = mp
			case reflect.Map:
				rawData = origin.(map[string]any)
			default:
				return newPipeErr[map[string]any](errors.New("origin 类型错误"))
			}

			// 注入_id
			mp := DefaultModelMap()
			mapper := &ModelCtxMapperPack{
				InjectData: mp,
			}
			err := mapper.Process(rawData)
			if err != nil {
				return newPipeErr[map[string]any](err)
			}

			_, err = db.Collection(params.ModelId).InsertOne(ctx, rawData)
			if err != nil {
				return newPipeErr[map[string]any](err)
			}

			return newPipeResult(rawData)
		},
	}
)
