package pipe

import (
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
)

type ModelDelConfig struct {
	QueryFilter *ut.QueryFull `json:"query_filter,omitempty"`
	ModelId     string        `json:"model_id,omitempty"`
	RowId       string        `json:"row_id,omitempty"`
}

var (
	// ModelDel 模型单条删除
	// 必传params ModelDelConfig 必传modelId和rowId
	// 必传db 为qmgo的qmgo.Database
	ModelDel = &RunnerContext[any, *ModelDelConfig, *qmgo.Database, any]{
		Key:  "model_ctx_del",
		Name: "模型单条删除",
		call: func(ctx iris.Context, origin any, params *ModelDelConfig, db *qmgo.Database, more ...any) *RunResp[any] {
			ft := params.QueryFilter
			if ft == nil {
				ft = new(ut.QueryFull)
			}
			ft.And = append(ft.And, &ut.Kov{
				Key:   ut.DefaultUidTag,
				Value: params.RowId,
			})

			var result = make(map[string]any)
			pipeline := ut.QueryToMongoPipeline(ft)
			err := db.Collection(params.ModelId).Aggregate(ctx, pipeline).One(&result)
			if err != nil {
				return NewPipeErr[any](err)
			}
			// 进行删除
			err = db.Collection(params.ModelId).Remove(ctx, bson.M{ut.DefaultUidTag: params.RowId})
			if err != nil {
				return NewPipeErr[any](err)
			}
			return NewPipeResult[any]("ok")
		},
	}
)
