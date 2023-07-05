package pipe

import (
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
)

// QueryParseConfig 所有的fields都是以json tag为准
type QueryParseConfig struct {
	SearchFields []string  `json:"search_fields,omitempty"`
	Pks          []*ut.Pk  `json:"pks,omitempty"`
	GeoKeys      []string  `json:"geo_keys,omitempty"` // 开启了geo的字段
	InjectAnd    []*ut.Kov `json:"inject_and,omitempty"`
	InjectOr     []*ut.Kov `json:"inject_or,omitempty"`
}

// 从传入的params中获取出 query and or page page_size sort等信息
// 从传入的参数中获取到外键 搜索 和 geo启用信息

var (
	QueryParse = &RunnerContext[any, *QueryParseConfig, any, *ut.QueryFull]{
		Key:  "model_ctx_query",
		Name: "模型query映射",
		call: func(ctx iris.Context, origin any, params *QueryParseConfig, db any, more ...any) *RunResp[*ut.QueryFull] {
			qs := ut.NewPruneCtxQuery()
			urlParams := ctx.URLParams()

			if params == nil {
				params = new(QueryParseConfig)
			}

			mapper, err := qs.PruneParse(urlParams, params.SearchFields, len(params.GeoKeys) >= 1)
			if err != nil {
				return newPipeErr[*ut.QueryFull](err)
			}

			if params.InjectAnd != nil {
				mapper.QueryParse.InsertOrReplaces("and", params.InjectAnd...)
			}
			if params.InjectOr != nil {
				mapper.QueryParse.InsertOrReplaces("or", params.InjectOr...)
			}
			mapper.Pks = params.Pks
			return newPipeResult(mapper)

		},
	}
)
