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
	QueryGetData = &RunnerContext[*ModelGetDataDep, *ModelGetData, *qmgo.Database, *ut.MongoFacetResult]{
		Key:  "query_get_data",
		Name: "query获取数据",
		call: func(ctx iris.Context, origin *ModelGetDataDep, params *ModelGetData, db *qmgo.Database, more ...any) *PipeRunResp[*ut.MongoFacetResult] {
			pipeline := ut.QueryToMongoPipeline(origin.Query)

			if params.Single {
				origin.Query.Page = 1
				origin.Query.PageSize = 1
			}

			var err error
			var all = new(ut.MongoFacetResult)

			if params.GetQueryCount && !params.Single {
				type inlineResp struct {
					Meta struct {
						Count int64 `json:"count"`
					}
					Data []map[string]any `json:"data,omitempty"`
				}
				var batch = make([]inlineResp, 0)
				err = db.Collection(origin.ModelId).Aggregate(ctx, pipeline).All(&batch)
				if len(batch) > 0 {
					all.Data = batch[0].Data
					all.Count = batch[0].Meta.Count
				}
			} else {
				var batch = make([]map[string]any, 0)
				err = db.Collection(origin.ModelId).Aggregate(ctx, pipeline).All(&batch)
				all.Data = batch
			}

			if err != nil {
				return newPipeErr[*ut.MongoFacetResult](err)
			}
			return newPipeResult(all)
		},
	}
)
