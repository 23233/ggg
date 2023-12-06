package pipe

import (
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/qiniu/qmgo"
)

type ModelGetData struct {
	Single        bool `json:"single,omitempty"`          // 仅获取单条 在single的情况下不会返回数量
	GetQueryCount bool `json:"get_query_count,omitempty"` // 返回匹配条数
}

type ModelGetDataDep struct {
	ModelId string        `json:"model_id,omitempty"`
	Query   *ut.QueryFull `json:"query,omitempty"`
}

var (
	// QueryGetData 通过模型解析出query获取内容
	// 必传origin ModelGetDataDep 中的modelId
	// 必传params ModelGetData 需要设定为是否为单条以及是否获取匹配条数
	// 必传db 为qmgo的Database
	QueryGetData = &RunnerContext[*ModelGetDataDep, *ModelGetData, *qmgo.Database, *ut.MongoFacetResult]{
		Key:  "query_get_data",
		Name: "query获取数据",
		call: func(ctx iris.Context, origin *ModelGetDataDep, params *ModelGetData, db *qmgo.Database, more ...any) *RunResp[*ut.MongoFacetResult] {
			if params.Single {
				origin.Query.Page = 1
				origin.Query.PageSize = 1
			}
			origin.Query.GetCount = params.GetQueryCount
			pipeline := ut.QueryToMongoPipeline(origin.Query)

			var err error
			var all = new(ut.MongoFacetResult)

			if params.Single {
				var result = make(map[string]any)
				err = db.Collection(origin.ModelId).Aggregate(ctx, pipeline).One(&result)
				all.Data = result
			} else {
				if params.GetQueryCount {
					type inlineResp struct {
						Meta struct {
							Count int64 `json:"count"`
						}
						Data []map[string]any `json:"data,omitempty"`
					}
					var batch = new(inlineResp)
					err = db.Collection(origin.ModelId).Aggregate(ctx, pipeline).One(&batch)
					all.Data = batch.Data
					all.Count = batch.Meta.Count

				} else {
					var batch = make([]map[string]any, 0)
					err = db.Collection(origin.ModelId).Aggregate(ctx, pipeline).All(&batch)
					all.Data = batch
				}
			}
			if err != nil {
				return NewPipeErr[*ut.MongoFacetResult](err)
			}
			return NewPipeResult(all)
		},
	}
)
