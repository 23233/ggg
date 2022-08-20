package mab

import (
	"encoding/json"
	"fmt"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"strings"
	"time"
)

func (rest *RestApi) ErrorResponse(err error, ctx iris.Context, msg ...string) {
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
	_ = ctx.JSON(iris.Map{
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
		rest.ErrorResponse(err, ctx)
		return
	}

	if sm.GetAllCheck != nil {
		pass, msg := sm.GetAllCheck(ctx, parse)
		if !pass {
			rest.ErrorResponse(nil, ctx, msg)
			return
		}
	}

	batch, result, err := ModelGetData(ctx, sm, rest.Cfg.Mdb, parse)
	if err != nil {
		rest.ErrorResponse(err, ctx, "查询数据出现错误")
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

	_ = ctx.JSON(result)
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
		rest.ErrorResponse(err, ctx)
		return
	}

	newData, err := ModelGetSingle(ctx, sm, rest.Cfg.Mdb, parse)
	if err != nil {
		rest.ErrorResponse(err, ctx, "查询数据失败")
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

	_ = ctx.JSON(newData)
}

// AddData 新增数据 使用json序列化 并且新增的是一个struct对象 所以可以使用qmgo的事件
// 注意事项 传入数据一定要符合`json`格式的预期 因为是直接通过 `json.Unmarshal` 结构到 struct
// 示例 Inline `bson:"inline"` 传入 {"key":""} 没有定义类型则直接传入inline的子元素
// 示范 Inline any `bson:",inline"` 传入 {"Inline":{"key":""}} 未设置json标签则取字段名称
// 示范 Inline any `json:"inline" bson:"inline"` 传入数据 {"inline":{"key":""}}
// 示范 JsonInLine any `json:"json_in_line,inline" bson:"inline"` 传入数据 {"json_in_line":{"key":""}}
// json tag的inline标签在Unmarshal时候似乎无作用
// 推荐 对于inline层面的数据 不要设置类型 直接引入 加入bson inline的tag标记即可
// eg: Inline `bson:",inline"`
func (rest *RestApi) AddData(ctx iris.Context) {
	sm := rest.PathGetModel(ctx.Path())
	if sm.CustomModel != nil {
		sm = sm.CustomModel(ctx, sm)
	}
	bodyRaw, err := ctx.GetBody()
	if err != nil {
		rest.ErrorResponse(err, ctx, "获取请求内容失败")
		return
	}

	// 主要作用为发现url必填参数是否传递
	var bodyParams map[string]any
	err = json.Unmarshal(bodyRaw, &bodyParams)
	if err != nil {
		rest.ErrorResponse(err, ctx, "请求体参数解析失败")
		return
	}

	reqModel := rest.newInterface(sm.Model)
	err = json.Unmarshal(bodyRaw, &reqModel)
	if err != nil {
		rest.ErrorResponse(err, ctx, "参数错误")
		return
	}

	// 如果有新增必须存在的参数
	if len(sm.PostMustFilters) > 0 {
		for k := range sm.PostMustFilters {
			if _, ok := bodyParams[k]; !ok {
				rest.ErrorResponse(nil, ctx, "必填参数缺失")
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
						rest.ErrorResponse(errors.New("检测到敏感词"), ctx, "请勿输入敏感词%s", firstWork)
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
			reflect.Indirect(reflect.ValueOf(reqModel)).Field(sm.privateIndex).Set(reflect.ValueOf(privateVal))
		}
	}

	// 如果需要把数据转化
	if sm.PostDataParse != nil {
		reqModel = sm.PostDataParse(ctx, reqModel)
	}

	if sm.PostDataCheck != nil {
		pass, msg := sm.PostDataCheck(ctx, reqModel)
		if !pass {
			rest.ErrorResponse(nil, ctx, msg)
			return
		}
	}

	aff, err := rest.Cfg.Mdb.Collection(sm.info.MapName).InsertOne(ctx, reqModel)
	if err != nil || len(aff.InsertedID.(primitive.ObjectID).Hex()) < 1 {
		rest.ErrorResponse(err, ctx, "新增数据失败")
		return
	}

	// 需要自定义返回
	if sm.PostResponseFunc != nil {
		reqModel = sm.PostResponseFunc(ctx, aff.InsertedID.(primitive.ObjectID).Hex(), reqModel)
	}

	_ = ctx.JSON(reqModel)
}

// EditData 修改数据 /{mid:string range(1,32)}
// 因为修改是按需传入变更的字段 而且会存在着直接传入 bson tag inline的情况 所以会先分析传入值 进行解析
// 如果 bson 标签有 inline 但是 json 标签有指定字段名时 传入json命名会自动结构到平级 传入下级会自动归类到json命名中
// 如果 json没有指定字段名 但是bson指定了时 传入的是字段名也就是大写 则会自动生成一个 bson字段名的元素
func (rest *RestApi) EditData(ctx iris.Context) {
	sm := rest.PathGetModel(ctx.Path())
	if sm.CustomModel != nil {
		sm = sm.CustomModel(ctx, sm)
	}

	bodyRaw, err := ctx.GetBody()
	if err != nil {
		rest.ErrorResponse(err, ctx, "获取请求内容失败")
		return
	}

	// 主要作用为发现url必填参数是否传递
	var pa bson.M
	err = json.Unmarshal(bodyRaw, &pa)
	if err != nil {
		rest.ErrorResponse(err, ctx, "请求体参数解析失败")
		return
	}

	for _, field := range sm.info.FieldList {
		if field.Kind == "struct" || field.Kind == "map" {
			// 如果 bson 标签有 inline 但是 json 标签有指定字段名
			if field.IsInline && len(field.JsonName) >= 1 {
				// 判断是否传入了
				if v, ok := pa[field.JsonName]; ok {
					// 如果有传入的话 提升到同级
					for kk, vv := range v.(map[string]any) {
						pa[kk] = vv
					}
				} else {
					// 没有传入判断下级是否传入
					var c = make(map[string]any)
					if len(field.Children) >= 1 {
						for _, child := range field.Children {
							if v, ok := pa[child.JsonName]; ok {
								c[child.JsonName] = v
							}
						}
						if len(c) >= 1 {
							pa[field.JsonName] = c
						}
					}

				}
				continue
			}

			// 如果json没有指定字段名 但是bson指定了
			if len(field.JsonName) < 1 && len(field.BsonName) >= 1 {
				// 如果传入的是字段名 则转换为bson的字段名
				if v, ok := pa[field.Name]; ok {
					pa[field.BsonName] = v
				}
			}

		}
	}

	// 实际的模型信息
	reqModel := rest.newInterface(sm.Model)
	b, _ := json.Marshal(&pa)
	err = json.Unmarshal(b, &reqModel)
	if err != nil {
		rest.ErrorResponse(nil, ctx, "参数错误")
		return
	}

	// 如果有修改必须存在的参数
	if len(sm.PutMustFilters) > 0 {
		for k := range sm.PutMustFilters {
			if _, ok := pa[k]; !ok {
				rest.ErrorResponse(nil, ctx, "必填参数缺失")
				return
			}
		}
	}

	// 进行敏感词验证
	if len(sm.sensitiveField) > 0 {
		for _, k := range sm.sensitiveField {
			if v, ok := pa[k]; ok {
				if val, ok := v.(string); ok {
					pass, firstWork := rest.runWordValid(val)
					if !pass {
						rest.ErrorResponse(errors.New("检测到敏感词"), ctx, "请勿输入敏感词%s", firstWork)
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
			reflect.Indirect(reflect.ValueOf(reqModel)).Field(sm.privateIndex).Set(reflect.ValueOf(privateVal))
		}
	}

	mid := ctx.Params().GetString("mid")
	if rest.Cfg.Generator {
		_, mid = rest.UriGetMid(ctx.Path())
	}
	objId, err := primitive.ObjectIDFromHex(mid)
	if err != nil {
		rest.ErrorResponse(err, ctx, "获取请求内容出错")
		return
	}
	privateValue := ctx.Values().Get(sm.PrivateContextKey)

	query := bson.M{"_id": objId}

	if sm.private {
		val, _ := sm.DisablePrivateMap["put"]
		if !val {
			query[sm.PrivateColName] = privateValue
		}
	}

	// 满足特殊需求的query
	if sm.PutQueryParse != nil {
		query = sm.PutQueryParse(ctx, mid, query, reqModel, privateValue)
	}

	// 获取旧数据
	data := rest.newInterface(sm.Model)
	err = rest.Cfg.Mdb.Collection(sm.info.MapName).Find(ctx, query).One(data)
	if err != nil {
		rest.ErrorResponse(err, ctx, "查询数据失败")
		return
	}

	// 把旧数据转换为bson
	dataBson := make(bson.M)
	b, _ = bson.Marshal(data)
	_ = bson.Unmarshal(b, &dataBson)

	// 把请求的数据也转换为bson
	reqBson := make(bson.M)
	b, _ = bson.Marshal(reqModel)
	_ = bson.Unmarshal(b, &reqBson)

	// 取不同 会存在嵌套struct整体更新的问题 逻辑上正常 暂不修改
	diff, _ := DiffBson(dataBson, reqBson, pa)

	// 如果没有什么不同 则直接返回
	if len(diff) < 1 {
		rest.ErrorResponse(err, ctx, "数据未产生变化")
		return
	}

	if sm.PutDataCheck != nil {
		pass, msg := sm.PutDataCheck(ctx, data, reqBson, pa, diff)
		if !pass {
			rest.ErrorResponse(nil, ctx, msg)
			return
		}
	}

	// 判断是否有更新时
	for _, field := range sm.info.FlatFields {
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
	if sm.PutDataParse != nil {
		diff = sm.PutDataParse(ctx, mid, diff)
	}

	err = rest.Cfg.Mdb.Collection(sm.info.MapName).UpdateId(ctx, objId, bson.M{"$set": diff})
	if err != nil {
		rest.ErrorResponse(err, ctx, "修改数据失败")
		return
	}

	// 更新缓存
	if sm.getSingleCacheTime() >= 1 {

		// 删除缓存 extra?
		rKey := genCacheKey(ctx.Request().RequestURI, sm.PrivateColName, fmt.Sprintf("%v", privateValue))
		success := rest.deleteAtCache(rKey)
		if !success {
			rest.Cfg.ErrorTrace(errors.New("cache delete fail"), "delete", "cache", "edit")
		}
	}

	if sm.PutResponseFunc != nil {
		diff = sm.PutResponseFunc(ctx, mid)
	}

	_ = ctx.JSON(diff)
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
		rest.ErrorResponse(err, ctx, "获取请求内容出错")
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
		rest.ErrorResponse(err, ctx, "预先获取数据失败")
		return
	}
	if sm.DeleteDataCheck != nil {
		pass, msg := sm.DeleteDataCheck(ctx, data)
		if !pass {
			rest.ErrorResponse(nil, ctx, msg)
			return
		}
	}

	// 再进行删除 不用担心先获取一次的性能消耗 根据统计 平均删除率不超过10%
	err = rest.Cfg.Mdb.Collection(sm.info.MapName).Remove(ctx, query)
	if err != nil {
		rest.ErrorResponse(err, ctx, "删除数据失败")
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
	_ = ctx.JSON(result)

}

// GetModelInfo 获取模型信息
func (rest *RestApi) GetModelInfo(ctx iris.Context) {
	modelName := ctx.Params().Get("modelName")

	// 获取模型
	for _, model := range rest.Cfg.Models {
		if model.info.MapName == modelName {
			if model.AllowGetInfo {
				_ = ctx.JSON(iris.Map{
					"info":  model.info,
					"empty": rest.newInterface(model.Model),
				})
				return
			}
			rest.ErrorResponse(errors.New("未授权获取信息"), ctx)
			return
		}
	}
	rest.ErrorResponse(errors.New("模型获取失败"), ctx)
	return
}

// 生成器模式下的验证中间件
func (rest *RestApi) generatorMiddleware(ctx iris.Context) {
	method := ctx.Method()
	modelPath, mid := rest.PathGetMid(method, ctx.Params().GetString("Model"))
	sm, err := rest.NameGetModel(modelPath)
	if err != nil {
		rest.ErrorResponse(err, ctx)
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
			ctx.AddHandler(rest.AddData)
			break
		case "put":
			if sm.getEditRate() != nil {
				ctx.AddHandler(LimitHandler(sm.getEditRate(), sm.RateErrorFunc))
			}
			ctx.AddHandler(rest.EditData)
			break
		case "delete":
			if sm.getDeleteRate() != nil {
				ctx.AddHandler(LimitHandler(sm.getDeleteRate(), sm.RateErrorFunc))
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
			_ = ctx.JSON(resp)
			return
		}

		ctx.Next()
	}
}
