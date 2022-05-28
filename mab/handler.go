package mab

import (
	"encoding/json"
	"fmt"
	"github.com/23233/ggg/sv"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"strings"
	"time"
)

// 错误返回
func fastError(err error, ctx iris.Context, msg ...string) {
	ctx.StatusCode(iris.StatusBadRequest)
	var m string
	if err == nil {
		m = "请求解析出错"
	} else {
		m = err.Error()
	}
	if len(msg) >= 1 {
		m = msg[0]
	}
	_, _ = ctx.JSON(iris.Map{
		"detail": m,
	})
	return
}

// GetAllFunc 获取所有
// page 控制页码 page_size 控制条数 最大均为100 100页 100条
// _last mid 大页码通用
// _o(asc) _od
// _s __在左右则为模糊 _s=__赵日天
// [字段名] 进行过滤 id=1 最长64位请注意 and关系
// _o_[字段名] 进行过滤 _o_id=2 最长64位 or关系
func (rest *RestApi) GetAllFunc(ctx iris.Context) {
	var err error
	sm := rest.PathGetModel(ctx.Path())
	if sm.CustomModel != nil {
		sm = sm.CustomModel(ctx, sm)
	}

	parse, err := CtxDataParse(ctx, sm, rest.Cfg.StructDelimiter)
	if err != nil {
		fastError(err, ctx)
		return
	}

	batch, result, err := ModelGetData(ctx, sm, rest.Cfg.Mdb, parse)
	if err != nil {
		fastError(err, ctx, "查询数据出现错误")
		return
	}

	// 如果需要自定义返回 把数据内容传过去
	if sm.GetAllResponseFunc != nil {
		result = sm.GetAllResponseFunc(ctx, result, batch)
	}

	// 判断是否开启了缓存
	if sm.getAllListCacheTime() >= 1 {
		var extraStr strings.Builder
		for _, v := range parse.ExtraBson {
			extraStr.WriteString(fmt.Sprintf("%s=%s", v.Key, v.Value))
		}

		// 生成key
		rKey := genCacheKey(ctx.Request().RequestURI, sm.PrivateColName, fmt.Sprintf("%v", ctx.Values().Get(sm.PrivateContextKey)), extraStr.String())
		err = rest.saveToCache(rKey, result, sm.getAllListCacheTime())
		if err != nil {
			rest.Cfg.ErrorTrace(err, "save_cache_error", "cache", "get(all)")
		}
	}

	_, _ = ctx.JSON(result)
}

// GetSingle 单个 /{mid:string range(1,32)}
func (rest *RestApi) GetSingle(ctx iris.Context) {
	sm := rest.PathGetModel(ctx.Path())
	if sm.CustomModel != nil {
		sm = sm.CustomModel(ctx, sm)
	}
	mid := ctx.Params().GetString("mid")
	if rest.Cfg.Generator {
		_, mid = rest.UriGetMid(ctx.Path())
	}

	parse, err := CtxSingleDataParse(ctx, sm, mid, rest.Cfg.StructDelimiter)
	if err != nil {
		fastError(err, ctx)
		return
	}

	newData, err := ModelGetSingle(ctx, sm, rest.Cfg.Mdb, parse)
	if err != nil {
		fastError(err, ctx, "查询数据失败")
		return
	}

	// 如果需要自定义返回 把数据内容传过去
	if sm.GetSingleResponseFunc != nil {
		newData = sm.GetSingleResponseFunc(ctx, newData)
	}

	if sm.getSingleCacheTime() >= 1 {
		var extraStr strings.Builder
		for _, v := range parse.ExtraBson {
			extraStr.WriteString(fmt.Sprintf("%s=%s", v.Key, v.Value))
		}
		// 生成key
		rKey := genCacheKey(ctx.Request().RequestURI, sm.PrivateColName, fmt.Sprintf("%v", ctx.Values().Get(sm.PrivateContextKey)), extraStr.String())
		err = rest.saveToCache(rKey, newData, sm.getSingleCacheTime())
		if err != nil {
			rest.Cfg.ErrorTrace(err, "save_cache_error", "cache", "get(single)")
		}
	}

	_, _ = ctx.JSON(newData)
}

// AddData 新增数据
func (rest *RestApi) AddData(ctx iris.Context) {
	ctx.RecordRequestBody(true)
	sm := rest.PathGetModel(ctx.Path())
	if sm.CustomModel != nil {
		sm = sm.CustomModel(ctx, sm)
	}
	// 主要作用为发现url必填参数是否传递
	var bodyParams map[string]interface{}
	_ = ctx.ReadBody(&bodyParams)
	req := ctx.Values().Get(sv.GlobalContextKey)
	if req == nil {
		fastError(errors.New("参数错误"), ctx)
		return
	}

	// 如果有新增必须存在的参数
	if len(sm.PostMustFilters) > 0 {
		for k := range sm.PostMustFilters {
			if _, ok := bodyParams[k]; !ok {
				fastError(nil, ctx, "必填参数缺失")
				return
			}
		}
	}

	// 进行敏感词验证
	if len(sm.sensitiveField) > 0 {
		for _, k := range sm.sensitiveField {
			if v, ok := bodyParams[k]; ok {
				if val, ok := v.(string); ok {
					pass, firstWork := rest.runWordValid(val)
					if !pass {
						fastError(errors.New("检测到敏感词"), ctx, "请勿输入敏感词%s", firstWork)
						return
					}
				}
			}
		}
	}

	if sm.private {
		val, _ := sm.DisablePrivateMap["post"]
		if !val {
			privateVal := ctx.Values().Get(sm.PrivateContextKey)
			reflect.Indirect(reflect.ValueOf(req)).Field(sm.privateIndex).Set(reflect.ValueOf(privateVal))
		}
	}

	// 如果需要把数据转化
	if sm.PostDataParse != nil {
		req = sm.PostDataParse(ctx, req)
	}

	aff, err := rest.Cfg.Mdb.Collection(sm.info.MapName).InsertOne(ctx, req)
	if err != nil || len(aff.InsertedID.(primitive.ObjectID).Hex()) < 1 {
		fastError(err, ctx, "新增数据失败")
		return
	}

	// 需要自定义返回
	if sm.PostResponseFunc != nil {
		req = sm.PostResponseFunc(ctx, aff.InsertedID.(primitive.ObjectID).Hex(), req)
	}

	_, _ = ctx.JSON(req)
}

// EditData 修改数据 /{mid:string range(1,32)}
func (rest *RestApi) EditData(ctx iris.Context) {
	model := rest.PathGetModel(ctx.Path())
	if model.CustomModel != nil {
		model = model.CustomModel(ctx, model)
	}

	// pa 就是传入的 params对象 仅做各类验证使用 有传入才会变更
	body, _ := ctx.GetBody()
	pa := make(bson.M)
	err := json.Unmarshal(body, &pa)
	if err != nil {
		fastError(err, ctx)
		return
	}

	// 这个是直接映射的模型 model 最后传入的关键
	reqModel := ctx.Values().Get(sv.GlobalContextKey)
	if reqModel == nil {
		fastError(errors.New("参数错误"), ctx)
		return
	}

	reqMap := make(bson.M)
	b, _ := bson.Marshal(reqModel)
	err = bson.Unmarshal(b, &reqMap)
	if err != nil {
		fastError(err, ctx)
		return
	}

	// 如果有修改必须存在的参数
	if len(model.PutMustFilters) > 0 {
		for k := range model.PutMustFilters {
			if _, ok := pa[k]; !ok {
				fastError(nil, ctx, "必填参数缺失")
				return
			}
		}
	}

	// 进行敏感词验证
	if len(model.sensitiveField) > 0 {
		for _, k := range model.sensitiveField {
			if v, ok := pa[k]; ok {
				if val, ok := v.(string); ok {
					pass, firstWork := rest.runWordValid(val)
					if !pass {
						fastError(errors.New("检测到敏感词"), ctx, "请勿输入敏感词%s", firstWork)
						return
					}
				}
			}
		}
	}

	if model.private {
		val, _ := model.DisablePrivateMap["post"]
		if !val {
			privateVal := ctx.Values().Get(model.PrivateContextKey)
			if _, ok := reqMap[model.PrivateColName]; ok {
				reflect.Indirect(reflect.ValueOf(reqModel)).Field(model.privateIndex).Set(reflect.ValueOf(privateVal))
			}
		}
	}

	mid := ctx.Params().GetString("mid")
	if rest.Cfg.Generator {
		_, mid = rest.UriGetMid(ctx.Path())
	}
	objId, err := primitive.ObjectIDFromHex(mid)
	if err != nil {
		fastError(err, ctx, "获取请求内容出错")
		return
	}
	privateValue := ctx.Values().Get(model.PrivateContextKey)

	data := rest.newInterface(model.Model)

	query := bson.M{"_id": objId}

	if model.private {
		val, _ := model.DisablePrivateMap["put"]
		if !val {
			query[model.PrivateColName] = privateValue
		}
	}

	// 满足特殊需求的query
	if model.PutQueryParse != nil {
		query = model.PutQueryParse(ctx, mid, query, reqModel, privateValue)
	}

	// 先获取这条数据
	err = rest.Cfg.Mdb.Collection(model.info.MapName).Find(ctx, query).One(data)
	if err != nil {
		fastError(err, ctx, "查询数据失败")
		return
	}

	dataBson := make(bson.M)
	b, _ = bson.Marshal(data)
	_ = bson.Unmarshal(b, &dataBson)

	// 取不同 会存在嵌套struct整体更新的问题 逻辑上正常 暂不修改
	diff, _ := DiffBson(dataBson, reqMap, pa)

	// 如果没有什么不同 则直接返回
	if len(diff) < 1 {
		fastError(err, ctx, "数据未产生变化")
		return
	}

	// 寻找inline内联 删除外层传递的参数
	// 新增不用这样处理 因为新增是使用struct device自动会进行处理
	for _, field := range model.info.FieldList {
		if v, ok := diff[field.MapName]; ok {
			if field.IsInline {
				for kk, vv := range v.(bson.M) {
					diff[kk] = vv
				}
				delete(diff, field.MapName)
			}
		}
	}

	// 判断是否有更新时
	for _, field := range model.info.FlatFields {
		if field.IsUpdated {
			// 判断参数中是否存在 存在则以参数中为准
			if v, ok := pa[field.MapName]; !ok {
				diff[field.MapName] = time.Now().Local()
			} else {
				diff[field.MapName], err = normalTimeParseBsonTime(v.(string))
			}
			break
		}
	}

	// 如果需要变更数据
	if model.PutDataParse != nil {
		diff = model.PutDataParse(ctx, mid, diff)
	}

	err = rest.Cfg.Mdb.Collection(model.info.MapName).UpdateId(ctx, objId, bson.M{"$set": diff})
	if err != nil {
		fastError(err, ctx, "修改数据失败")
		return
	}

	// 更新缓存
	if model.getSingleCacheTime() >= 1 {

		// 删除缓存 extra?
		rKey := genCacheKey(ctx.Request().RequestURI, model.PrivateColName, fmt.Sprintf("%v", privateValue))
		success := rest.deleteAtCache(rKey)
		if !success {
			rest.Cfg.ErrorTrace(errors.New("cache delete fail"), "delete", "cache", "edit")
		}
	}

	if model.PutResponseFunc != nil {
		diff = model.PutResponseFunc(ctx, mid)
	}

	_, _ = ctx.JSON(diff)
}

// DeleteData 删除数据 /{mid:string range(1,32)}
func (rest *RestApi) DeleteData(ctx iris.Context) {
	sm := rest.PathGetModel(ctx.Path())
	if sm.CustomModel != nil {
		sm = sm.CustomModel(ctx, sm)
	}
	privateValue := ctx.Values().Get(sm.PrivateContextKey)
	mid := ctx.Params().GetString("mid")
	if rest.Cfg.Generator {
		_, mid = rest.UriGetMid(ctx.Path())
	}
	objId, err := primitive.ObjectIDFromHex(mid)
	if err != nil {
		fastError(err, ctx, "获取请求内容出错")
		return
	}

	query := bson.M{"_id": objId}

	if sm.private {
		val, _ := sm.DisablePrivateMap["delete"]
		if !val {
			query[sm.PrivateColName] = privateValue
		}
	}

	data := bson.M{}

	// 先获取一下数据
	err = rest.Cfg.Mdb.Collection(sm.info.MapName).Find(ctx, query).One(&data)
	if err != nil {
		fastError(err, ctx, "预先获取数据失败")
		return
	}

	// 再进行删除 不用担心先获取一次的性能消耗 根据统计 平均删除率不超过10%
	err = rest.Cfg.Mdb.Collection(sm.info.MapName).Remove(ctx, query)
	if err != nil {
		fastError(err, ctx, "删除数据失败")
		return
	}
	result := iris.Map{"id": mid}
	if sm.DeleteResponseFunc != nil {
		result = sm.DeleteResponseFunc(ctx, mid, data, result)
	}

	// 删除缓存
	if sm.getSingleCacheTime() >= 1 {
		// 删除缓存 extra?
		rKey := genCacheKey(ctx.Request().RequestURI, sm.PrivateColName, fmt.Sprintf("%v", privateValue))
		success := rest.deleteAtCache(rKey)
		if !success {
			rest.Cfg.ErrorTrace(errors.New("cache delete fail"), "delete", "cache", "delete")
		}
	}
	_, _ = ctx.JSON(result)

}

// GetModelInfo 获取模型信息
func (rest *RestApi) GetModelInfo(ctx iris.Context) {
	modelName := ctx.Params().Get("modelName")

	// 获取模型
	for _, model := range rest.Cfg.Models {
		if model.info.MapName == modelName {
			if model.AllowGetInfo {
				_, _ = ctx.JSON(iris.Map{
					"info":  model.info,
					"empty": rest.newInterface(model.Model),
				})
				return
			}
			fastError(errors.New("未授权获取信息"), ctx)
			return
		}
	}
	fastError(errors.New("模型获取失败"), ctx)
	return
}

// 生成器模式下的验证中间件
func (rest *RestApi) generatorMiddleware(ctx iris.Context) {
	method := ctx.Method()
	modelPath, mid := rest.PathGetMid(method, ctx.Params().GetString("Model"))
	sm, err := rest.NameGetModel(modelPath)
	if err != nil {
		fastError(err, ctx)
		return
	}

	var parseMethod string
	if method == "GET" {
		if len(mid) > 1 {
			parseMethod = "get(single)"
		} else {
			parseMethod = "get(all)"
		}
	} else {
		parseMethod = strings.ToLower(method)
	}

	if isContain(sm.getMethods(), parseMethod) {
		switch parseMethod {
		case "get(all)":
			if sm.getAllRate() != nil {
				ctx.AddHandler(LimitHandler(sm.getAllRate(), sm.RateErrorFunc))
			}
			// cache
			if sm.getAllListCacheTime() > 0 {
				ctx.AddHandler(rest.getCacheMiddleware("list"))
			}
			ctx.AddHandler(rest.GetAllFunc)
			break
		case "get(single)":
			if sm.getSingleRate() != nil {
				ctx.AddHandler(LimitHandler(sm.getSingleRate(), sm.RateErrorFunc))
			}
			if sm.getSingleCacheTime() > 0 {
				ctx.AddHandler(rest.getCacheMiddleware("single"))
			}
			ctx.AddHandler(rest.GetSingle)
			break
		case "post":
			if sm.getAddRate() != nil {
				ctx.AddHandler(LimitHandler(sm.getAddRate(), sm.RateErrorFunc))
			}
			if sm.PostValidator != nil {
				ctx.AddHandler(sv.Run(sm.PostValidator))
			} else {
				ctx.AddHandler(sv.Run(sm.Model, "json"))
			}
			ctx.AddHandler(rest.AddData)
			break
		case "put":
			if sm.getEditRate() != nil {
				ctx.AddHandler(LimitHandler(sm.getEditRate(), sm.RateErrorFunc))
			}
			// 判断是否有自定义验证器
			if sm.PutValidator != nil {
				ctx.AddHandler(sv.Run(sm.PutValidator))
			} else {
				ctx.AddHandler(sv.Run(sm.Model, "json"))
			}
			ctx.AddHandler(rest.EditData)
			break
		case "delete":
			if sm.getDeleteRate() != nil {
				ctx.AddHandler(LimitHandler(sm.getDeleteRate(), sm.RateErrorFunc))
			}
			// 判断是否有自定义验证器
			if sm.DeleteValidator != nil {
				ctx.AddHandler(sv.Run(sm.DeleteValidator))
			}
			ctx.AddHandler(rest.DeleteData)
			break
		}
	} else {
		ctx.NotFound()
		return
	}

	ctx.Next()
}

// 获取数据的中间件
func (rest *RestApi) getCacheMiddleware(from string) iris.Handler {
	return func(ctx iris.Context) {
		model := rest.PathGetModel(ctx.Path())
		// 判断header中 Cache-control
		cacheHeader := ctx.GetHeader("Cache-control")
		if cacheHeader == "no-cache" {
			ctx.Next()
			return
		}
		privateValue := ctx.Values().Get(model.PrivateContextKey)
		var extraStr strings.Builder
		var extraParams map[string]interface{}

		if from == "list" {
			if model.GetAllExtraFilters != nil {
				extraParams = model.GetAllExtraFilters(ctx)
			}
		} else {
			if model.GetSingleExtraFilters != nil {
				extraParams = model.GetSingleExtraFilters(ctx)
			}
		}
		for k, v := range extraParams {
			extraStr.WriteString(fmt.Sprintf("%s=%s", k, v))
		}
		// 获取参数 生成key
		rKey := genCacheKey(ctx.Request().RequestURI, model.PrivateColName, fmt.Sprintf("%v", privateValue), extraStr.String())

		resp, err := restCache.Get(rKey)
		if err == nil {
			ctx.Header("is_cache", "1")
			_, _ = ctx.JSON(resp)
			return
		}

		ctx.Next()
	}
}
