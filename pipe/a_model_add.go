package pipe

import (
	jsoniter "github.com/json-iterator/go"
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

func StructToMap(s any) (map[string]any, error) {
	data, err := jsoniter.Marshal(s)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	err = jsoniter.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	// 遍历map，删除值为nil或空map的键值对
	for k, v := range m {
		if v == nil {
			delete(m, k)
		} else if reflect.TypeOf(v).Kind() == reflect.Map {
			subMap, ok := v.(map[string]interface{})
			if ok && len(subMap) == 0 {
				delete(m, k)
			}
		}
	}

	return m, nil
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
				mp, err := StructToMap(origin)
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
