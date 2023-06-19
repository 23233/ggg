package pipe

import (
	"github.com/kataras/iris/v12"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	ModelAdd = &RunnerContext[any, *ModelCtxAddConfig, *qmgo.Database, any]{
		Key:  "model_ctx_add",
		Name: "模型json新增",
		call: func(ctx iris.Context, origin any, params *ModelCtxAddConfig, db *qmgo.Database, more ...any) *RunResp[any] {
			if origin == nil {
				return newPipeErr[any](PipeDepNotFound)
			}
			// 注入_id
			mp := DefaultModelMap()
			mapper := &ModelCtxMapperPack{
				InjectData: mp,
			}
			err := mapper.Process(origin)
			if err != nil {
				return newPipeErr[any](err)
			}
			// 进行数据验证?

			_, err = db.Collection(params.ModelId).InsertOne(ctx, mp)
			if err != nil {
				return newPipeErr[any](err)
			}

			return newPipeResult(origin)
		},
	}
)
