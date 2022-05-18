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

var operators = []string{"eq", "gt", "gte", "lt", "lte", "ne", "in", "nin"}

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
			n = strings.Replace(k, prefix, "", 1)
		}
		// 判断是否拥有操作符
		hasOp, n, op := containOperators(k)

		// 传入的字段名是 map 或者 fullName
		if field.ParamsKey == n {
			// 为了安全 值长度限制一下
			if len(v) > 64 {
				break
			}
			var item bson.E

			// 赋值未过类型检测机制 则跳过
			val, err := typeGetVal(v, field.Types)
			if err != nil {
				break
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
				item.Value = bson.M{"$" + op: val}
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
func cmpQuery(or, filter, search, private, extra []bson.E, lastMid primitive.ObjectID) bson.D {
	// bson.D[bson.E{"$and":bson.A[bson.D{bson.E,bson.E}]},{"$or": bson.A{ bson.D{{}}, bson.D{{}}}]
	r := bson.D{}

	if !lastMid.IsZero() {
		r = append(r, bson.E{
			Key: "_id",
			Value: bson.M{
				"$gt": lastMid,
			},
		})
	}

	andItem := bson.D{}
	if len(filter) > 0 {
		for _, e := range filter {
			andItem = append(andItem, e)
		}
	}
	if len(private) > 0 {
		for _, e := range private {
			andItem = append(andItem, e)
		}
	}
	if len(extra) > 0 {
		for _, e := range extra {
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

	if len(or) > 0 || len(search) > 0 {
		// {"$or": bson.A{ bson.D{{}}, bson.D{{}}}
		orItem := bson.A{}
		if len(search) > 0 {
			for _, e := range search {
				orItem = append(orItem, bson.D{e})
			}
		}
		if len(or) > 0 {
			for _, e := range or {
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
func fkCmpQuery(match bson.D, lookup []bson.D, page, pageSize int64, descField, orderBy string) []bson.D {
	// 这里顺序特别重要 一定不能随意变更顺序 必须是match在前 lookup中间
	pipeline := make([]bson.D, 0)
	if len(match) > 0 {
		pipeline = append(pipeline, bson.D{{"$match", match}})
	}
	pipeline = append(pipeline, lookup...)

	// 解析出sort 顺序在limit skip之前
	sort := bson.D{}
	if len(descField) > 0 {
		sort = append(sort, bson.E{Key: descField, Value: -1})
	}
	if len(orderBy) > 0 {
		sort = append(sort, bson.E{Key: orderBy, Value: 1})
	}
	if len(sort) > 0 {
		pipeline = append(pipeline, bson.D{{"$sort", sort}})
	}
	skip := (page - 1) * pageSize
	if skip > 0 {
		pipeline = append(pipeline, bson.D{{"$skip", skip}})
	}
	pipeline = append(pipeline, bson.D{{"$limit", pageSize}})

	return pipeline
}

// geoQuery 地理位置获取参数
func geoQuery(match bson.D, lookup []bson.D, page, pageSize int64, lng, lat float64, maxDistance, minDistance int64, descField, orderBy string) []bson.D {
	// 这里顺序特别重要 一定不能随意变更顺序 geo信息必须在最前
	pipeline := make([]bson.D, 0)

	var near = bson.M{
		"near": bson.M{
			"type":        "Point",
			"coordinates": []float64{lng, lat},
		},
		"distanceField": "_distance",
		"spherical":     true,
	}
	if maxDistance >= 1 {
		near["maxDistance"] = maxDistance
	}
	if minDistance >= 1 {
		near["minDistance"] = minDistance
	}
	if len(match) > 0 {
		near["query"] = match
	}

	geoNear := bson.D{{"$geoNear", near}}
	pipeline = append(pipeline, geoNear)

	pipeline = append(pipeline, lookup...)

	// 解析出sort 顺序在limit skip之前
	// 在geo模式下尽量不要使用sort 应该使用默认的距离返回模式
	sort := bson.D{}
	if len(descField) > 0 {
		sort = append(sort, bson.E{Key: descField, Value: -1})
	}
	if len(orderBy) > 0 {
		sort = append(sort, bson.E{Key: orderBy, Value: 1})
	}
	if len(sort) > 0 {
		pipeline = append(pipeline, bson.D{{"$sort", sort}})
	}

	pipeline = append(pipeline, bson.D{{"$skip", (page - 1) * pageSize}})
	pipeline = append(pipeline, bson.D{{"$limit", pageSize}})

	return pipeline
}

func even(number int) bool {
	return number%2 == 0
}

// DiffBson 两个bson.m 对比 获取异同
func DiffBson(o bson.M, n bson.M, jsonData bson.M) (diff bson.M, eq bson.M) {
	diff = make(bson.M)
	eq = make(bson.M)
	for k, v := range o {
		// 先判断拥有同样的key
		val, ok := n[k]
		if ok {
			// 变更必须存在于已传入了这个参数
			if _, ok := jsonData[k]; ok {
				// 判断值是否相同
				if cmp.Equal(v, val) {
					eq[k] = val
				} else {
					diff[k] = val
				}
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
		d.Bson = strings.Split(field.Tag.Get("bson"), ",")
		d.JsonTag = strings.Split(field.Tag.Get("json"), ",")
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
