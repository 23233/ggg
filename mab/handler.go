package mab

import (
	"encoding/json"
	"fmt"
	"github.com/23233/ggg/sv"
	"github.com/devfeel/mapper"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"strconv"
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
	page := ctx.URLParamInt64Default("page", 1)
	maxCount, maxSize := sm.getPage()
	if page > maxCount {
		page = maxCount
	}
	pageSize := ctx.URLParamInt64Default("page_size", 10)
	if pageSize > maxSize {
		pageSize = maxSize
	}

	var urlParamsMap = ctx.URLParams()
	if sm.InjectParams != nil {
		urlParamsMap = sm.InjectParams(ctx)
	}

	// 从url中解析出filter
	filterList, orList := filterMatch(urlParamsMap, sm.info.FlatFields, rest.Cfg.StructDelimiter)
	// 判断必传参数是否存在
	if len(sm.GetAllMustFilters) > 0 {
		for k := range sm.GetAllMustFilters {
			has := false
			for _, f := range append(filterList, orList...) {
				if k == f.Key {
					has = true
					break
				}
			}
			if !has {
				fastError(nil, ctx, "必填参数缺失")
				return
			}
		}
	}

	// 解析出有没有geo信息
	geoStr := ctx.URLParam("_g")
	geoMax, _ := ctx.URLParamInt64("_gmax")
	geoMin, _ := ctx.URLParamInt64("_gmin")
	var lng, lat float64
	var hasGeo = len(geoStr) >= 1
	if hasGeo {
		if !strings.Contains(geoStr, ",") {
			fastError(errors.New("_g 参数格式错误"), ctx)
			return
		}
		geoList := strings.Split(geoStr, ",")
		if len(geoList) != 2 {
			fastError(errors.New("_g 参数格式解析错误"), ctx)
			return
		}
		lng, err = strconv.ParseFloat(geoList[0], 64)
		if err != nil {
			fastError(err, ctx)
			return
		}
		lat, err = strconv.ParseFloat(geoList[1], 64)
		if err != nil {
			fastError(err, ctx)
			return
		}
	}

	// 最后的id
	lastMid := ctx.URLParam("_last")
	var lastObj primitive.ObjectID
	if len(lastMid) > 0 {
		lastObj, err = primitive.ObjectIDFromHex(lastMid)
		if err != nil {
			fastError(err, ctx, "last mid 解析失败")
			return
		}
	}

	// 解析出order by
	descField := ctx.URLParam("_od")
	orderBy := ctx.URLParam("_o")
	sortList := make([]string, 0, 2)
	if len(descField) > 0 {
		sortList = append(sortList, "-"+descField)
	}
	if len(orderBy) > 0 {
		sortList = append(sortList, orderBy)
	}

	// 如果有额外附加字段
	extraBson := make([]bson.E, 0)
	if sm.GetAllExtraFilters != nil {
		extraMap := sm.GetAllExtraFilters(ctx)
		for k, v := range extraMap {
			extraBson = append(extraBson, bson.E{
				Key:   k,
				Value: v,
			})
		}
	}

	// 匹配出搜索
	searchStr := ctx.URLParam("_s")
	searchBson := make([]bson.E, 0)
	if len(searchStr) >= 1 {
		if len(sm.searchFields) < 1 {
			fastError(errors.New("搜索功能未启用"), ctx)
			return
		}
		v := strings.ReplaceAll(searchStr, "__", "")
		objId, err := primitive.ObjectIDFromHex(v)
		if err == nil {
			if sm.MustSearch {
				for _, info := range sm.info.FieldList {
					if info.IsObjId {
						orList = append(orList, bson.E{
							Key:   info.MapName,
							Value: objId,
						})
					}
				}
				orList = append(orList, bson.E{Key: "_id", Value: objId})
			}
		} else {
			for _, field := range sm.searchFields {
				if strings.HasPrefix(searchStr, "__") && strings.HasSuffix(searchStr, "__") {
					searchBson = append(searchBson, bson.E{
						Key: field.MapName,
						Value: bson.D{
							{"$regex", primitive.Regex{Pattern: v, Options: "i"}},
						},
					})
					continue
				}
				// 如果是前匹配
				if strings.HasPrefix(searchStr, "__") {
					searchBson = append(searchBson, bson.E{
						Key: field.MapName,
						Value: bson.D{
							{"$regex", primitive.Regex{Pattern: "^" + v, Options: "i"}},
						},
					})
					continue
				}
				// 如果是后匹配
				if strings.HasSuffix(searchStr, "__") {
					searchBson = append(searchBson, bson.E{
						Key: field.MapName,
						Value: bson.D{
							{"$regex", primitive.Regex{Pattern: v + "$", Options: "i"}},
						},
					})
					continue
				}
			}
		}
	}

	// 私密参数
	privateBson := make([]bson.E, 0)
	if sm.private {
		val, _ := sm.DisablePrivateMap["get(all)"]
		if !val {
			privateValue := ctx.Values().Get(sm.PrivateContextKey)
			privateBson = append(privateBson, bson.E{
				Key:   sm.PrivateColName,
				Value: privateValue,
			})
		}
	}

	//判断是否附加外键
	lookup := make([]bson.D, 0)
	if sm.Pk != nil {
		lookup = sm.Pk()
	}
	// 批量查询 必须为map 因为如果有外键会新增字段 无法映射到struct上面去
	// 如果判断外键存在切换的话 会存在返回数据不一致的问题
	batch := make([]bson.M, 0)

	// 组合出查询条件
	query := cmpQuery(orList, filterList, searchBson, privateBson, extraBson, lastObj)

	// 如果有geo 进入geo分支
	if hasGeo {
		pipeline := geoQuery(query, lookup, page, pageSize, lng, lat, geoMax, geoMin, descField, orderBy)
		err = rest.Cfg.Mdb.Collection(sm.info.MapName).Aggregate(ctx, pipeline).All(&batch)
	} else {
		// 如果是包含外键的
		if len(lookup) > 0 {
			pipeline := fkCmpQuery(query, lookup, page, pageSize, descField, orderBy)
			err = rest.Cfg.Mdb.Collection(sm.info.MapName).Aggregate(ctx, pipeline).All(&batch)
		} else {
			err = rest.Cfg.Mdb.Collection(sm.info.MapName).Find(ctx, query).Limit(pageSize).Skip((page - 1) * pageSize).Sort(sortList...).All(&batch)
		}
	}

	if err != nil {
		fastError(err, ctx, "查询出现错误")
		return
	}

	result := iris.Map{
		"page_size": pageSize,
		"page":      page,
		"data":      batch,
		"has_more":  (int64(len(batch)) - pageSize) >= 0,
	}

	if sm.ShowCount {
		count, err := rest.Cfg.Mdb.Collection(sm.info.MapName).Find(ctx, query).Count()
		if err != nil {
			fastError(err, ctx, "获取数量失败")
			return
		}
		result["count"] = count
	}

	if sm.ShowDocCount {
		mColl, _ := rest.Cfg.Mdb.Collection(sm.info.MapName).CloneCollection()
		allCount, err := mColl.EstimatedDocumentCount(ctx)
		if err != nil {
			fastError(err, ctx, "获取文档总数量失败")
			return
		}
		result["doc_count"] = allCount
	}

	if len(descField) >= 1 {
		result["desc_field"] = descField
	}
	if len(orderBy) >= 1 {
		result["order"] = orderBy
	}
	if len(filterList) >= 1 {
		result["filter"] = filterList
	}
	if len(orList) >= 1 {
		result["or"] = orList
	}
	if len(searchStr) >= 1 {
		result["search"] = searchStr
	}

	// 如果需要自定义返回 把数据内容传过去
	if sm.GetAllResponseFunc != nil {
		result = sm.GetAllResponseFunc(ctx, result, batch)
	}

	// 判断是否开启了缓存
	if sm.getAllListCacheTime() >= 1 {
		var extraStr strings.Builder
		for _, v := range extraBson {
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
	privateBson := make([]bson.E, 0)

	// 私密字段
	if sm.private {
		val, _ := sm.DisablePrivateMap["get(single)"]
		if !val {
			privateValue := ctx.Values().Get(sm.PrivateContextKey)
			privateBson = append(privateBson, bson.E{
				Key:   sm.PrivateColName,
				Value: privateValue,
			})
		}
	}

	// 如果有额外附加字段
	extraBson := make([]bson.E, 0)
	if sm.GetSingleExtraFilters != nil {
		extraMap := sm.GetSingleExtraFilters(ctx)
		for k, v := range extraMap {
			extraBson = append(extraBson, bson.E{
				Key:   k,
				Value: v,
			})
		}
	}
	//
	var urlParamsMap = ctx.URLParams()
	if sm.InjectParams != nil {
		urlParamsMap = sm.InjectParams(ctx)
	}
	// 从url中解析出filter 一般是不会传的
	filterList, orList := filterMatch(urlParamsMap, sm.info.FlatFields, rest.Cfg.StructDelimiter)
	// 判断必传参数是否存在
	if len(sm.GetSingleMustFilters) > 0 {
		for k := range sm.GetSingleMustFilters {
			has := false
			for i, f := range append(filterList, orList...) {
				if even(i) {
					if k == f.Key {
						has = true
						break
					}
				}
			}
			if !has {
				fastError(nil, ctx, "参数错误")
				return
			}
		}
	}

	objId, _ := primitive.ObjectIDFromHex(mid)

	filterList = append(filterList, bson.E{
		Key:   "_id",
		Value: objId,
	})

	// 一样必须为bson.M 如果为struct 则无法映射外键数据
	var newData = bson.M{}

	var err error
	//判断是否附加外键
	lookup := make([]bson.D, 0)
	if sm.Pk != nil {
		lookup = sm.Pk()
	}
	query := cmpQuery(orList, filterList, nil, privateBson, extraBson, primitive.NilObjectID)

	if len(lookup) > 0 {
		pipeline := fkCmpQuery(query, lookup, 1, 1, "", "")
		err = rest.Cfg.Mdb.Collection(sm.info.MapName).Aggregate(ctx, pipeline).One(newData)
	} else {
		err = rest.Cfg.Mdb.Collection(sm.info.MapName).Find(ctx, query).One(newData)
	}

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
		for _, v := range extraBson {
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

	body, _ := ctx.GetBody()
	var pa map[string]interface{}
	_ = json.Unmarshal(body, &pa)

	req := ctx.Values().Get(sv.GlobalContextKey)
	if req == nil {
		fastError(errors.New("参数错误"), ctx)
		return
	}

	reqMap := make(map[string]interface{})
	err := mapper.Mapper(req, &reqMap)
	if err != nil {
		fastError(err, ctx)
		return
	}

	// 如果有新增必须存在的参数
	if len(model.PostMustFilters) > 0 {
		for k := range model.PostMustFilters {
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
				reflect.Indirect(reflect.ValueOf(req)).Field(model.privateIndex).Set(reflect.ValueOf(privateVal))
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
		query = model.PutQueryParse(ctx, mid, query, req, privateValue)
	}

	// 先获取这条数据
	err = rest.Cfg.Mdb.Collection(model.info.MapName).Find(ctx, query).One(data)
	if err != nil {
		fastError(err, ctx, "查询数据失败")
		return
	}

	// 取不同 会存在嵌套struct整体更新的问题 逻辑上正常 暂不修改
	diff, _ := DiffBson(data, req, pa)

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
				diff[field.MapName] = v
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
