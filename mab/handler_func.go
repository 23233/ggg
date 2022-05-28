package mab

import (
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strconv"
	"strings"
)

type CtxGeoInfo struct {
	Lng    float64
	Lat    float64
	GeoMax int64
	GeoMin int64
}

type CtxGetDataParse struct {
	OrBson      []bson.E // or
	FilterBson  []bson.E // and
	PrivateBson []bson.E // 私密
	ExtraBson   []bson.E // 额外附加
	SortDesc    string   // 降序
	SortAsc     string   // 升序
	SortList    []string // order by
	LastMid     primitive.ObjectID
	HasGeo      bool       // 是否包含geo信息
	GeoInfo     CtxGeoInfo // geo信息
	Search      string     // 搜索词
	SearchBson  []bson.E   // 对应的搜索bson
	Pk          []bson.D   // 外键
	Page        int64
	PageSize    int64
}

func CtxDataParse(ctx iris.Context, sm *SingleModel, delimiter string) (*CtxGetDataParse, error) {
	var err error
	var r = new(CtxGetDataParse)

	r.Page = ctx.URLParamInt64Default("page", 1)
	maxCount, maxSize := sm.getPage()
	if r.Page > maxCount {
		r.Page = maxCount
	}
	r.PageSize = ctx.URLParamInt64Default("page_size", 10)
	if r.PageSize > maxSize {
		r.PageSize = maxSize
	}

	var urlParamsMap = ctx.URLParams()
	if sm.InjectParams != nil {
		urlParamsMap = sm.InjectParams(ctx)
	}

	// 从url中解析出filter
	filterList, orList := filterMatch(urlParamsMap, sm.info.FlatFields, delimiter)
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
				return nil, errors.New("必填参数缺失")
			}
		}
	}

	r.FilterBson = filterList
	r.OrBson = orList

	// 解析出有没有geo信息
	geoStr := ctx.URLParam("_g")
	geoMax, _ := ctx.URLParamInt64("_gmax")
	geoMin, _ := ctx.URLParamInt64("_gmin")
	var lng, lat float64
	var hasGeo = len(geoStr) >= 1
	if hasGeo {
		if !strings.Contains(geoStr, ",") {
			return nil, errors.New("_g 参数格式错误")
		}
		geoList := strings.Split(geoStr, ",")
		if len(geoList) != 2 {
			return nil, errors.New("_g 参数格式解析错误")
		}
		lng, err = strconv.ParseFloat(geoList[0], 64)
		if err != nil {
			return nil, err
		}
		lat, err = strconv.ParseFloat(geoList[1], 64)
		if err != nil {
			return nil, err

		}
		r.GeoInfo = CtxGeoInfo{
			Lng:    lng,
			Lat:    lat,
			GeoMax: geoMax,
			GeoMin: geoMin,
		}
	}

	r.HasGeo = hasGeo

	// 最后的id
	lastMid := ctx.URLParam("_last")
	var lastObj primitive.ObjectID
	if len(lastMid) > 0 {
		lastObj, err = primitive.ObjectIDFromHex(lastMid)
		if err != nil {
			return nil, errors.Wrap(err, "last mid 解析失败")
		}
	}

	r.LastMid = lastObj

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

	r.SortDesc = descField
	r.SortAsc = orderBy
	r.SortList = sortList

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

	r.ExtraBson = extraBson

	// 匹配出搜索
	searchStr := ctx.URLParam("_s")
	searchBson := make([]bson.E, 0)
	if len(searchStr) >= 1 {
		if len(sm.searchFields) < 1 {
			return nil, errors.New("搜索功能未启用")
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

	r.Search = searchStr
	r.SearchBson = searchBson

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

	r.PrivateBson = privateBson

	//判断是否附加外键
	lookup := make([]bson.D, 0)
	if sm.Pk != nil {
		lookup = sm.Pk()
	}

	r.Pk = lookup

	return r, nil
}

func ModelGetData(ctx iris.Context, sm *SingleModel, mdb *qmgo.Database, parse *CtxGetDataParse) ([]bson.M, iris.Map, error) {
	var err error

	// 批量查询 必须为map 因为如果有外键会新增字段 无法映射到struct上面去
	// 如果判断外键存在切换的话 会存在返回数据不一致的问题
	batch := make([]bson.M, 0)

	// 组合出查询条件
	query := cmpQuery(parse.OrBson, parse.FilterBson, parse.SearchBson, parse.PrivateBson, parse.ExtraBson, parse.LastMid)

	// 如果有geo 进入geo分支
	if parse.HasGeo {
		pipeline := geoQuery(query, parse.Pk, parse.Page, parse.PageSize, parse.GeoInfo.Lng, parse.GeoInfo.Lat, parse.GeoInfo.GeoMax, parse.GeoInfo.GeoMin, parse.SortDesc, parse.SortAsc)
		err = mdb.Collection(sm.info.MapName).Aggregate(ctx, pipeline).All(&batch)
	} else {
		// 如果是包含外键的
		if len(parse.Pk) > 0 {
			pipeline := fkCmpQuery(query, parse.Pk, parse.Page, parse.PageSize, parse.SortDesc, parse.SortAsc)
			err = mdb.Collection(sm.info.MapName).Aggregate(ctx, pipeline).All(&batch)
		} else {
			err = mdb.Collection(sm.info.MapName).Find(ctx, query).Limit(parse.PageSize).Skip((parse.Page - 1) * parse.PageSize).Sort(parse.SortList...).All(&batch)
		}
	}
	if err != nil {
		return nil, nil, errors.Wrap(err, "获取数据失败")
	}

	result := iris.Map{
		"page_size": parse.PageSize,
		"page":      parse.Page,
		"data":      batch,
		"has_more":  (int64(len(batch)) - parse.PageSize) >= 0,
	}

	if sm.ShowCount {
		count, err := mdb.Collection(sm.info.MapName).Find(ctx, query).Count()
		if err != nil {
			return batch, nil, errors.Wrap(err, "获取数量失败")
		}
		result["count"] = count
	}

	if sm.ShowDocCount {
		mColl, _ := mdb.Collection(sm.info.MapName).CloneCollection()
		allCount, err := mColl.EstimatedDocumentCount(ctx)
		if err != nil {
			return batch, nil, errors.Wrap(err, "获取文档总数量失败")
		}
		result["doc_count"] = allCount
	}

	if len(parse.SortDesc) >= 1 {
		result["desc_field"] = parse.SortDesc
	}
	if len(parse.SortAsc) >= 1 {
		result["order"] = parse.SortAsc
	}
	if len(parse.FilterBson) >= 1 {
		result["filter"] = parse.FilterBson
	}
	if len(parse.OrBson) >= 1 {
		result["or"] = parse.OrBson
	}
	if len(parse.Search) >= 1 {
		result["search"] = parse.Search
	}

	return batch, result, err
}

func CtxSingleDataParse(ctx iris.Context, sm *SingleModel, mid string, delimiter string) (*CtxGetDataParse, error) {
	var r = new(CtxGetDataParse)

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

	r.PrivateBson = privateBson

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

	r.ExtraBson = extraBson

	var urlParamsMap = ctx.URLParams()
	if sm.InjectParams != nil {
		urlParamsMap = sm.InjectParams(ctx)
	}
	// 从url中解析出filter 一般是不会传的
	filterList, orList := filterMatch(urlParamsMap, sm.info.FlatFields, delimiter)
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
				return nil, errors.New("必传参数错误")
			}
		}
	}

	objId, err := primitive.ObjectIDFromHex(mid)
	if err != nil {
		return nil, errors.Wrap(err, "解析mid错误")
	}

	filterList = append(filterList, bson.E{
		Key:   "_id",
		Value: objId,
	})

	r.FilterBson = filterList
	r.OrBson = orList

	//判断是否附加外键
	lookup := make([]bson.D, 0)
	if sm.Pk != nil {
		lookup = sm.Pk()
	}
	r.Pk = lookup

	return r, nil
}

func ModelGetSingle(ctx iris.Context, sm *SingleModel, mdb *qmgo.Database, parse *CtxGetDataParse) (bson.M, error) {
	// 一样必须为bson.M 如果为struct 则无法映射外键数据
	var newData = bson.M{}
	query := cmpQuery(parse.OrBson, parse.FilterBson, nil, parse.PrivateBson, parse.ExtraBson, primitive.NilObjectID)

	var err error
	// 如果是包含外键的
	if len(parse.Pk) > 0 {
		pipeline := fkCmpQuery(query, parse.Pk, 1, 1, parse.SortDesc, parse.SortAsc)
		err = mdb.Collection(sm.info.MapName).Aggregate(ctx, pipeline).One(&newData)
	} else {
		err = mdb.Collection(sm.info.MapName).Find(ctx, query).One(&newData)
	}
	if err != nil {
		return nil, err
	}

	return newData, nil
}
