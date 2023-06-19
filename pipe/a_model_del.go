package pipe

import (
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
)

type ModelDelConfig struct {
	ModelId string `json:"model_id,omitempty"`
	RowId   string `json:"row_id,omitempty"`
}

var (
	ModelDel = &RunnerContext[any, *ModelDelConfig, *qmgo.Database, any]{
		Key:  "model_ctx_del",
		Name: "模型单条删除",
		call: func(ctx iris.Context, origin any, params *ModelDelConfig, db *qmgo.Database, more ...any) *PipeRunResp[any] {
			var result = make(map[string]any)
			err := db.Collection(params.ModelId).Find(ctx, bson.M{ut.DefaultUidTag: params.RowId}).One(&result)
			if err != nil {
				return newPipeErr[any](err)
			}
			// 进行删除
			err = db.Collection(params.ModelId).Remove(ctx, bson.M{ut.DefaultUidTag: params.RowId})
			if err != nil {
				return newPipeErr[any](err)
			}
			return newPipeResult[any]("ok")
		},
	}
)
