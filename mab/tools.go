package mab

import (
	"github.com/23233/ggg/ut"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// 字符串转换成bool
func parseBool(str string) (bool, error) {
	switch str {
	case "1", "t", "T", "true", "TRUE", "True":
		return true, nil
	case "0", "f", "F", "false", "FALSE", "False":
		return false, nil
	}
	return false, errors.New("解析出错")
}
func IsNum(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func isContain(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

var operators = []string{"eq", "gt", "gte", "lt", "lte", "ne", "in", "nin", "exists", "null"}

// 操作后缀的分隔符
var operatorsSep = "_"

// containOperators 判断key是否包含操作列表
func containOperators(k string) (bool, string, string) {
	for _, operator := range operators {
		simple := operatorsSep + operator
		full := simple + operatorsSep
		if strings.HasSuffix(k, simple) {
			kk := strings.TrimSuffix(k, simple)
			return true, kk, operator
		} else if strings.HasSuffix(k, full) {
			kk := strings.TrimSuffix(k, full)
			return true, kk, operator
		}
	}
	return false, k, ""
}

func typeGetVal(v string, fieldType string) (any, error) {
	var val any = v
	var err error

	// 检测value是否非string
	switch fieldType {
	case "int", "int8", "int16", "int32", "int64", "time.Duration":
		val, err = strconv.ParseInt(v, 10, 64)
		break
	case "uint", "uint8", "uint16", "uint32", "uint64":
		val, err = strconv.ParseUint(v, 10, 64)
		break
	case "float32", "float64":
		val, err = strconv.ParseFloat(v, 64)
		break
	case "bool":
		val, err = parseBool(v)
		break
	case "time.Time":
		// 最主要解决的是mongodb插入的时间是UTC时间
		// 所以需要进行比较的时间一定要是UTC格式
		val, err = normalTimeParseBsonTime(val.(string))
		break
	case "primitive.ObjectID":
		val, err = primitive.ObjectIDFromHex(v)
		break
	}
	return val, err
}

// 从字段信息中 匹配参数
func paramsMatch(k, v string, prefix string, fields []StructInfo, structDelimiter string) []bson.E {
	result := make([]bson.E, 0)
	for _, field := range fields {
		v := strings.Trim(v, " ") // 先去掉前后的空格
		n := k                    // 去掉前缀
		if len(prefix) > 0 {
			n = n[len(prefix):]
		}
		// 判断是否拥有操作符
		hasOp, n, op := containOperators(n)

		// 传入的字段名是 map 或者 fullName
		if field.ParamsKey == n {
			// 为了安全 值长度限制一下
			if len(v) > 64 {
				break
			}
			var item bson.E
			var val any
			var err error

			// 赋值未过类型检测机制 则跳过
			// 若操作为bool则不用校验类型
			if op != "null" && op != "exists" {
				val, err = typeGetVal(v, field.Types)
				if err != nil {
					break
				}
			}

			// 包含struct分隔符就是fullName 默认value是string
			if strings.Contains(n, structDelimiter) {
				item = bson.E{
					Key:   strings.ReplaceAll(field.ParamsKey, structDelimiter, "."),
					Value: val,
				}
			} else {
				item = bson.E{
					Key:   field.MapName,
					Value: val,
				}
			}
			// 判断是否有操作后缀
			if hasOp {
				// 判断是包含或不包含
				if op == "in" || op == "nin" {
					inList := strings.Split(v, ",")
					val = inList
					// 如果字段是数组
					if field.Kind == "slice" {
						var result = make([]interface{}, 0, len(inList))
						for _, s := range inList {
							v, err := typeGetVal(s, strings.TrimPrefix(field.Types, "[]"))
							if err == nil {
								result = append(result, v)
							}
						}
						val = result
					}
				}

				// 为查询字段是否存在
				if op == "exists" {
					if v == "0" || v == "false" || len(v) < 1 {
						val = false
					} else {
						val = true
					}
				}

				item.Value = bson.M{"$" + op: val}

				// 检查字段内容是否存在
				if op == "null" {
					if v == "0" || v == "false" || len(v) < 1 {
						var m = []any{nil, ""}
						item.Value = bson.M{"$nin": m}
					} else {
						item.Value = bson.M{"$ne": nil}
					}
				}

			}

			result = append(result, item)
		}
	}
	return result
}

// filterMatch 从url params上解析出mongo必备的数据结构
func filterMatch(fullParams map[string]string, fields []StructInfo, structDelimiter string) ([]bson.E, []bson.E) {
	filter := make([]bson.E, 0)
	or := make([]bson.E, 0)
	filterPrefix := ""
	orPrefix := "_o_"
	if len(fullParams) > 0 {
		for k, v := range fullParams {
			if strings.HasPrefix(k, orPrefix) {
				or = append(or, paramsMatch(k, v, orPrefix, fields, structDelimiter)...)
			} else {
				filter = append(filter, paramsMatch(k, v, filterPrefix, fields, structDelimiter)...)
			}
		}
	}
	return filter, or
}

// 组合参数
func cmpQuery(parse *CtxGetDataParse) bson.D {
	// bson.D[bson.E{"$and":bson.A[bson.D{bson.E,bson.E}]},{"$or": bson.A{ bson.D{{}}, bson.D{{}}}]
	r := bson.D{}

	if !parse.LastMid.IsZero() {
		r = append(r, bson.E{
			Key: "_id",
			Value: bson.M{
				"$" + parse.LastSort: parse.LastMid,
			},
		})
	}

	andItem := bson.D{}
	if len(parse.FilterBson) > 0 {
		for _, e := range parse.FilterBson {
			andItem = append(andItem, e)
		}
	}
	if len(parse.PrivateBson) > 0 {
		for _, e := range parse.PrivateBson {
			andItem = append(andItem, e)
		}
	}
	if len(parse.ExtraBson) > 0 {
		for _, e := range parse.ExtraBson {
			andItem = append(andItem, e)
		}
	}
	if len(andItem) > 0 {
		// bson.A[bson.D{bson.E,bson.E}]
		r = append(r, bson.E{
			Key:   "$and",
			Value: bson.A{andItem},
		})
	}

	if len(parse.OrBson) > 0 || len(parse.SearchBson) > 0 {
		// {"$or": bson.A{ bson.D{{}}, bson.D{{}}}
		orItem := bson.A{}
		if len(parse.SearchBson) > 0 {
			for _, e := range parse.SearchBson {
				orItem = append(orItem, bson.D{e})
			}
		}
		if len(parse.OrBson) > 0 {
			for _, e := range parse.OrBson {
				orItem = append(orItem, bson.D{e})
			}
		}

		r = append(r, bson.E{
			Key:   "$or",
			Value: orItem,
		})
	}

	return r
}

// 组合包含外键的参数
func fkCmpQuery(match bson.D, parse *CtxGetDataParse) []bson.D {
	// 这里顺序特别重要 一定不能随意变更顺序 必须是match在前 lookup中间
	pipeline := make([]bson.D, 0)
	if len(match) > 0 {
		pipeline = append(pipeline, bson.D{{"$match", match}})
	}
	pipeline = append(pipeline, parse.Pk...)

	// 解析出sort 顺序在limit skip之前
	sort := bson.D{}
	if len(parse.SortDesc) > 0 {
		for _, s := range parse.SortDesc {
			sort = append(sort, bson.E{Key: s, Value: -1})
		}
	}
	if len(parse.SortAsc) > 0 {
		for _, s := range parse.SortAsc {
			sort = append(sort, bson.E{Key: s, Value: 1})
		}
	}
	if len(sort) > 0 {
		pipeline = append(pipeline, bson.D{{"$sort", sort}})
	}
	skip := (parse.Page - 1) * parse.PageSize
	if skip > 0 {
		pipeline = append(pipeline, bson.D{{"$skip", skip}})
	}
	pipeline = append(pipeline, bson.D{{"$limit", parse.PageSize}})

	return pipeline
}

// geoQuery 地理位置获取参数
func geoQuery(match bson.D, parse *CtxGetDataParse) []bson.D {
	// 这里顺序特别重要 一定不能随意变更顺序 geo信息必须在最前
	pipeline := make([]bson.D, 0)

	var near = bson.M{
		"near": bson.M{
			"type":        "Point",
			"coordinates": []float64{parse.GeoInfo.Lng, parse.GeoInfo.Lat},
		},
		"distanceField": "_distance",
		"spherical":     true,
	}
	if parse.GeoInfo.GeoMax >= 1 {
		near["maxDistance"] = parse.GeoInfo.GeoMax
	}
	if parse.GeoInfo.GeoMin >= 1 {
		near["minDistance"] = parse.GeoInfo.GeoMin
	}
	if len(match) > 0 {
		near["query"] = match
	}

	geoNear := bson.D{{"$geoNear", near}}
	pipeline = append(pipeline, geoNear)

	pipeline = append(pipeline, parse.Pk...)

	// 解析出sort 顺序在limit skip之前
	// 在geo模式下尽量不要使用sort 应该使用默认的距离返回模式
	sort := bson.D{}
	if len(parse.SortDesc) > 0 {
		for _, s := range parse.SortDesc {
			sort = append(sort, bson.E{Key: s, Value: -1})
		}
	}
	if len(parse.SortAsc) > 0 {
		for _, s := range parse.SortAsc {
			sort = append(sort, bson.E{Key: s, Value: 1})
		}
	}
	if len(sort) > 0 {
		pipeline = append(pipeline, bson.D{{"$sort", sort}})
	}

	pipeline = append(pipeline, bson.D{{"$skip", (parse.Page - 1) * parse.PageSize}})
	pipeline = append(pipeline, bson.D{{"$limit", parse.PageSize}})

	return pipeline
}

func even(number int) bool {
	return number%2 == 0
}

// DiffBson 两个bson.m 对比 获取异同
func DiffBson(oldData bson.M, wantData bson.M, reqSendData bson.M) (diff bson.M, eq bson.M) {
	diff = make(bson.M)
	eq = make(bson.M)

	// 遍历传入的参数
	for k, v := range reqSendData {
		val, ok := wantData[k]
		if !ok {
			// 在旧数据存在说明字段存在 但是新提交的数据可能为false这种会过滤的数据
			_, ok = oldData[k]
			if !ok {
				continue
			}
			// 旧值中存在的话 则把val直接赋值为发送上来的数据类型
			val = v
		}
		ov, ok := oldData[k]
		if !ok {
			// 在原值中未找到的话
			diff[k] = val
		} else {
			// 在原值中找到 判断是否一致
			if cmp.Equal(ov, val) {
				eq[k] = val
			} else {
				diff[k] = val
			}
		}
	}

	return
}

func UTCTrans(utcTime string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05.000+08:00", utcTime)
}

func TableNameReflectFieldsAndTypes(table interface{}, structDelimiter string) []StructInfo {
	return TableNameGetNestedStructMaps(reflect.TypeOf(table), "", "", "", structDelimiter)
}

func TableNameGetNestedStructMaps(r reflect.Type, parentStructName, parentMapName string, parentLevel string, structDelimiter string) []StructInfo {
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}
	if r.Kind() != reflect.Struct {
		return nil
	}

	childrenSkipType := []string{"time.Time", "primitive.ObjectID"}

	result := make([]StructInfo, 0)
	for i := 0; i < r.NumField(); i++ {
		field := r.Field(i)
		var d StructInfo

		d.Name = field.Name
		d.Kind = field.Type.Kind().String()
		d.Comment = field.Tag.Get("comment")
		d.CustomTag = field.Tag.Get("mab")
		d.ValidateTag = field.Tag.Get("validate")
		d.DescTag = field.Tag.Get("desc")
		d.Bson = strings.Split(field.Tag.Get("bson"), ",")
		if len(d.Bson) >= 1 {
			d.BsonName = d.Bson[0]
			if d.BsonName == "," || d.BsonName == "omitempty" || d.BsonName == "inline" {
				d.BsonName = ""
			}
		}
		d.JsonTag = strings.Split(field.Tag.Get("json"), ",")
		if len(d.JsonTag) >= 1 {
			d.JsonName = d.JsonTag[0]
			if d.JsonName == "," || d.JsonName == "omitempty" {
				d.JsonName = ""
			}
		}
		d.Index = i
		if len(parentLevel) > 0 {
			d.Level = strings.Join([]string{parentLevel, "-", strconv.Itoa(i)}, "")
		} else {
			d.Level = strconv.Itoa(i)
		}
		if isContain(d.Bson, "inline") {
			d.IsInline = true
		}
		d.Types = field.Type.String()
		// 有bson取bson 没有就转Snake
		if len(d.Bson) >= 1 {
			d.MapName = d.Bson[0]
		}
		// todo 这里有争议 有人希望是蛇形 有人希望保持字段名 暂定蛇形吧
		if len(d.MapName) < 1 {
			d.MapName = ut.STN(field.Name)
		}

		if len(parentStructName) > 0 {
			d.FullName = parentStructName + structDelimiter + d.Name
		}
		if len(parentMapName) > 0 {
			d.FullMapName = parentMapName + structDelimiter + d.MapName
			d.ParamsKey = parentMapName + structDelimiter + d.MapName
		} else {
			d.ParamsKey = d.MapName
		}

		// 判断是否是主键 使用约定式
		switch field.Name {
		case "Id":
			d.IsPk = true
			break
		case "UpdateAt":
			d.IsUpdated = true
			break
		case "CreateAt":
			d.IsCreated = true
			break
		case "DeleteAt":
			d.IsDeleted = true
			break
		case "DefaultField":
			d.IsDefaultWrap = true
			break
		}

		if d.Types == "primitive.ObjectID" {
			d.IsObjId = true
		}

		if d.Types == "time.Time" {
			d.IsTime = true
		}

		// 先获取children
		if field.Type.Kind() == reflect.Struct {
			if !isContain(childrenSkipType, field.Type.String()) {
				if d.IsInline {
					d.Children = TableNameGetNestedStructMaps(field.Type, d.Name, "", d.Level, structDelimiter)
				} else {
					d.Children = TableNameGetNestedStructMaps(field.Type, d.Name, d.MapName, d.Level, structDelimiter)
				}
			}
			d.ChildrenKind = field.Type.Kind().String()
		} else if field.Type.Kind() == reflect.Slice {
			elem := field.Type.Elem()
			if elem.Kind() == reflect.Struct {
				d.Children = TableNameGetNestedStructMaps(field.Type.Elem(), d.Name, d.MapName, d.Level, structDelimiter)
			}
			d.ChildrenKind = elem.Kind().String()
		}

		// 判断是否是geo
		if len(d.Children) == 2 {
			for _, child := range d.Children {
				if child.Name == "Coordinates" {
					d.IsGeo = true
					break
				}
			}
		}

		result = append(result, d)
	}
	return result
}

// 拍平字段
func flatField(fields []StructInfo) []StructInfo {
	result := make([]StructInfo, 0)
	for _, field := range fields {
		if len(field.Children) > 0 {
			result = append(result, flatField(field.Children)...)
		}
		result = append(result, field)
	}
	return result
}

// 时间解析
func normalTimeParseBsonTime(s string) (primitive.DateTime, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse("2006-01-02 15:04:05", s)
	}
	return primitive.NewDateTimeFromTime(t), err
}
