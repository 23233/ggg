package scene

import (
	"encoding/json"
	"github.com/23233/ggg/pipe"
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/qiniu/qmgo"
)

var (
	FuncGetAll = func(ctx iris.Context, self ISceneModelItem) *pipe.RunResp[any] {
		mapper := self.GetMapper()
		if mapper == nil {
			return pipe.NewPipeErrMsg[any]("mapper为空", nil)
		}
		queryParams := pipe.QueryParseConfig{
			UrlParams: mapper.Query,
		}

		// 核验是否必传上一次分页id
		if self.GetExtra().MustLastId {
			if _, ok := mapper.Query["_last"]; !ok {
				return pipe.NewPipeErrMsg[any]("必传参数缺失", nil)
			}
		}

		q, err := getQueryParse(ctx, self.GetContextInjectFilter(), self.GetFilterInjects(), queryParams)
		if err != nil {
			return pipe.NewPipeErrMsg[any]("解析注入内容失败", err)
		}

		// 解析query
		resp := pipe.QueryParse.Run(ctx, nil, q, nil)
		if resp.Err != nil {
			return pipe.NewPipeErrMsg[any]("解析查询参数失败", resp.Err)
		}
		if resp.Result.Page < 1 {
			resp.Result.Page = 1
		}
		if resp.Result.Page > 100 {
			resp.Result.Page = 100
		}
		if resp.Result.PageSize <= 0 {
			resp.Result.PageSize = 10
		}
		if resp.Result.PageSize > 100 {
			resp.Result.PageSize = 100
		}
		// 默认按照更新时间倒序
		if resp.Result.SortAsc == nil && resp.Result.SortDesc == nil {
			resp.Result.SortDesc = append(resp.Result.SortDesc, "update_at")
		}
		if len(resp.Result.SortAsc) < 1 && len(resp.Result.SortDesc) < 1 {
			resp.Result.SortDesc = append(resp.Result.SortDesc, "update_at")
		}

		var getCount = self.GetExtra().GetCount

		// 获取数据
		dataResp := pipe.QueryGetData.Run(ctx,
			&pipe.ModelGetDataDep{
				ModelId: self.GetTableName(),
				Query:   resp.Result,
			},
			&pipe.ModelGetData{
				Single:        false,
				GetQueryCount: getCount,
			},
			self.GetDb())
		if dataResp.Err != nil {
			if dataResp.Err != qmgo.ErrNoSuchDocuments {
				return pipe.NewPipeErrMsg[any]("查询出内容失败", resp.Err)
			}
		}

		var result = new(SchemaGetResp)
		result.MongoFacetResult = dataResp.Result
		result.Page = resp.Result.Page
		result.PageSize = resp.Result.PageSize
		result.Filters = resp.Result.QueryParse
		result.Sorts = resp.Result.BaseSort
		return pipe.NewPipeResult[any](result)
	}
	FuncGetSingle = func(ctx iris.Context, self ISceneModelItem) *pipe.RunResp[any] {
		mapper := self.GetMapper()
		if mapper == nil {
			return pipe.NewPipeErrMsg[any]("mapper为空", nil)
		}

		uid := mapper.GetUid()
		if len(uid) < 1 {
			return pipe.NewPipeErrMsg[any]("获取参数uid失败", nil)
		}
		queryParams := pipe.QueryParseConfig{}
		queryParams.InjectAnd = append(queryParams.InjectAnd, &ut.Kov{
			Key:   ut.DefaultUidTag,
			Value: uid,
		})
		q, err := getQueryParse(ctx, self.GetContextInjectFilter(), self.GetFilterInjects(), queryParams)
		if err != nil {
			return pipe.NewPipeErrMsg[any]("解析注入内容失败", err)
		}

		// 解析query
		resp := pipe.QueryParse.Run(ctx, nil, q, nil)
		if resp.Err != nil {
			return pipe.NewPipeErrMsg[any]("解析查询参数失败", err)
		}

		// 获取数据
		dataResp := pipe.QueryGetData.Run(ctx,
			&pipe.ModelGetDataDep{
				ModelId: self.GetTableName(),
				Query:   resp.Result,
			},
			&pipe.ModelGetData{
				Single: true,
			},
			self.GetDb())
		if dataResp.Err != nil {
			return pipe.NewPipeErrMsg[any]("查询出数据失败", dataResp.Err)
		}

		// 未获取到
		if dataResp.Result.Data == nil {
			return pipe.NewPipeErrMsg[any]("获取数据不存在", nil)
		}

		return pipe.NewPipeResult(dataResp.Result.Data)
	}
	FuncPostAdd = func(ctx iris.Context, self ISceneModelItem) *pipe.RunResp[any] {
		mapper := self.GetMapper()
		if mapper == nil {
			return pipe.NewPipeErrMsg[any]("mapper为空", nil)
		}
		if len(mapper.Body) < 1 {
			return pipe.NewPipeErrMsg[any]("内容体为空", nil)
		}

		// 注入的数据
		injectQuery, err := ParseInject(ctx, self.GetContextInjectFilter()...)
		if err != nil {
			return pipe.NewPipeErrMsg[any]("解析注入数据失败", err)
		}
		injectQuery = append(injectQuery, self.GetFilterInjects()...)
		var injectData = make(map[string]any, len(injectQuery))
		for _, kov := range injectQuery {
			injectData[kov.Key] = kov.Value
		}

		// 把body字符串序列化成map 进行 schema的核验
		var bodyMap = make(map[string]any)
		err = json.Unmarshal([]byte(mapper.Body), &bodyMap)
		if err != nil {
			return pipe.NewPipeErrMsg[any]("解析body参数体失败", err)
		}

		// 把注入数据注入到map中
		for k, v := range injectData {
			bodyMap[k] = v
		}

		// 进行json schema的核验
		resp := pipe.SchemaValid.Run(ctx, bodyMap, &pipe.SchemaValidConfig{
			Schema: self.GetSchema(),
		}, nil)
		if resp.Err != nil {
			return pipe.NewPipeErr[any](resp.Err)
		}

		// 把字符串序列化成body
		var newInst = self.GetRawNew()
		err = json.Unmarshal([]byte(mapper.Body), newInst)
		if err != nil {
			return pipe.NewPipeErrMsg[any]("解析body参数体失败", err)
		}
		// 构建参数
		// 通过模型去序列化body 可以防止一些无效的数据注入
		injectResult := pipe.ModelMapper.Run(ctx, newInst, &pipe.ModelCtxMapperPack{
			InjectData: injectData,
		}, nil)
		if injectResult.Err != nil {
			return pipe.NewPipeErrMsg[any]("数据核验失败", injectResult.Err)
		}

		// 进行新增
		insertResult := pipe.ModelAdd.Run(ctx, injectResult.Result, &pipe.ModelCtxAddConfig{ModelId: self.GetTableName()}, self.GetDb())
		if insertResult.Err != nil {
			return pipe.NewPipeErrMsg[any]("新增失败", insertResult.Err)
		}

		return pipe.NewPipeResult[any](insertResult.Result)

	}
	FuncEdit = func(ctx iris.Context, self ISceneModelItem) *pipe.RunResp[any] {
		mapper := self.GetMapper()
		if mapper == nil {
			return pipe.NewPipeErrMsg[any]("mapper为空", nil)
		}
		if len(mapper.Body) < 1 {
			return pipe.NewPipeErrMsg[any]("内容体为空", nil)

		}
		uid := mapper.GetUid()
		if len(uid) < 1 {
			return pipe.NewPipeErrMsg[any]("获取关键参数失败", nil)
		}

		// 把body序列化成map 好进行diff比对
		var bodyMap = make(map[string]any)
		err := json.Unmarshal([]byte(mapper.Body), &bodyMap)
		if err != nil {
			return pipe.NewPipeErrMsg[any]("解析body参数体失败", err)
		}

		// 解析query
		injectQuery, err := ParseInject(ctx, self.GetContextInjectFilter()...)
		if err != nil {
			return pipe.NewPipeErrMsg[any]("解析注入数据失败", err)

		}
		queryFilter := new(ut.QueryFull)
		queryFilter.QueryParse.InsertOrReplaces("and", injectQuery...)
		queryFilter.QueryParse.InsertOrReplaces("and", self.GetFilterInjects()...)

		var params = pipe.ModelPutConfig{
			QueryFilter: queryFilter,
			ModelId:     self.GetTableName(),
			RowId:       uid,
			BodyMap:     bodyMap,
		}
		// 把字符串序列化成body
		var newInst = self.GetRawNew()
		err = json.Unmarshal([]byte(mapper.Body), newInst)
		if err != nil {
			return pipe.NewPipeErrMsg[any]("解析body参数体失败", err)
		}

		resp := pipe.ModelPut.Run(ctx, newInst, &params, self.GetDb())
		if resp.Err != nil {
			return pipe.NewPipeErrMsg[any]("修改失败", resp.Err)
		}
		return pipe.NewPipeResult[any](resp.Result)
	}
	FuncDelete = func(ctx iris.Context, self ISceneModelItem) *pipe.RunResp[any] {
		mapper := self.GetMapper()
		if mapper == nil {
			return pipe.NewPipeErrMsg[any]("mapper为空", nil)
		}
		uid := mapper.GetUid()
		if len(uid) < 1 {
			return pipe.NewPipeErrMsg[any]("获取关键参数失败", nil)
		}

		// 解析query
		injectQuery, err := ParseInject(ctx, self.GetContextInjectFilter()...)
		if err != nil {
			return pipe.NewPipeErrMsg[any]("解析注入数据失败", err)
		}
		queryFilter := new(ut.QueryFull)
		if queryFilter.QueryParse == nil {
			queryFilter.QueryParse = new(ut.QueryParse)
		}
		queryFilter.QueryParse.InsertOrReplaces("and", injectQuery...)
		queryFilter.QueryParse.InsertOrReplaces("and", self.GetFilterInjects()...)
		var params = pipe.ModelDelConfig{
			QueryFilter: queryFilter,
			ModelId:     self.GetTableName(),
			RowId:       uid,
		}
		resp := pipe.ModelDel.Run(ctx, nil, &params, self.GetDb())
		if resp.Err != nil {
			return pipe.NewPipeErrMsg[any]("删除失败", resp.Err)
		}
		return pipe.NewPipeResult[any](resp.Result)
	}
)

func getQueryParse(ctx iris.Context, contextInject []ContextValueInject, Injects []*ut.Kov, queryParams pipe.QueryParseConfig) (*pipe.QueryParseConfig, error) {
	injectQuery, err := ParseInject(ctx, contextInject...)
	if err != nil {
		return nil, err
	}
	if queryParams.InjectAnd == nil {
		queryParams.InjectAnd = make([]*ut.Kov, 0)
	}
	queryParams.InjectAnd = append(queryParams.InjectAnd, injectQuery...)

	if Injects != nil && len(Injects) >= 1 {
		queryParams.InjectAnd = append(queryParams.InjectAnd, Injects...)
	}
	return &queryParams, nil
}
