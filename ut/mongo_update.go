package ut

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"image"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// StructConverter 作用在于struct中如果需要更新的时候 不要使用bson.M来手动设置字段名
// 这样又没有编辑器提示 字段变更时也很难同步 可以采取这种方式
// 特点是会自动去除0值的字段最后只保留有值的产出bson.M
// 那在某些情况下 会从db中取出存储的哪一行 然后跟当前的进行对比
// 这时候就使用 diff 的模式 对比两个struct中的不同项进行变更
// 还有某些情况下 需要从body中取出更新项 需要使用 pipe里的diff算法
type StructConverter struct {
	CustomIsZero map[reflect.Type]func(reflect.Value, reflect.StructField) bool
	SpecialTypes []reflect.Type // 特殊结构体类型列表
}

func (sc *StructConverter) InitSpecialTypes() {
	sc.SpecialTypes = []reflect.Type{
		reflect.TypeOf(time.Time{}),
		reflect.TypeOf(net.IP{}),
		reflect.TypeOf(net.IPNet{}),
		reflect.TypeOf(url.URL{}),
		reflect.TypeOf(regexp.Regexp{}),
		reflect.TypeOf(os.File{}),
		reflect.TypeOf((*os.FileInfo)(nil)).Elem(),
		reflect.TypeOf(big.Int{}),
		reflect.TypeOf(big.Float{}),
		reflect.TypeOf(big.Rat{}),
		reflect.TypeOf(zip.Reader{}),
		reflect.TypeOf(zip.Writer{}),
		reflect.TypeOf(http.Request{}),
		reflect.TypeOf(http.Response{}),
		reflect.TypeOf((*image.Image)(nil)).Elem(),
		reflect.TypeOf(csv.Reader{}),
		reflect.TypeOf(csv.Writer{}),
		reflect.TypeOf(json.Decoder{}),
		reflect.TypeOf(json.Encoder{}),
		reflect.TypeOf(xml.Decoder{}),
		reflect.TypeOf(xml.Encoder{}),
	}
}

func (sc *StructConverter) isSpecialStructType(t reflect.Type) bool {
	// 检查给定类型是否在特殊类型列表中
	for _, specialType := range sc.SpecialTypes {
		if t == specialType {
			return true
		}
	}
	return false
}
func (sc *StructConverter) IsZero(v reflect.Value, field reflect.StructField) bool {
	if customIsZero, ok := sc.CustomIsZero[v.Type()]; ok && customIsZero(v, field) {
		return true
	}

	zeroValue := reflect.Zero(v.Type()).Interface()
	return cmp.Equal(v.Interface(), zeroValue)
}

func (sc *StructConverter) AddCustomIsZero(value interface{}, fn func(reflect.Value, reflect.StructField) bool) {
	typ := reflect.TypeOf(value)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if sc.CustomIsZero == nil {
		sc.CustomIsZero = make(map[reflect.Type]func(reflect.Value, reflect.StructField) bool)
	}
	sc.CustomIsZero[typ] = fn
}

func (sc *StructConverter) StructToBsonM(v interface{}, freeze ...string) (bson.M, error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, errors.New("输入必须是结构体或结构体指针")
	}

	freezeMap := make(map[string]bool)
	for _, path := range freeze {
		freezeMap[path] = true
	}

	return sc.structToMap(val, freezeMap, ""), nil
}

func (sc *StructConverter) structToMap(val reflect.Value, freezeMap map[string]bool, prefix string) bson.M {
	result := bson.M{}
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldType := typ.Field(i)
		// 如果字段未导出，则跳过
		if !fieldType.IsExported() {
			continue
		}

		if sc.IsZero(fieldVal, fieldType) {
			continue
		}

		bsonTag := fieldType.Tag.Get("bson")
		if bsonTag == "-" {
			continue
		}

		bsonTagParts := strings.Split(bsonTag, ",")
		bsonTag = bsonTagParts[0]

		isInline := strings.Contains(bsonTag, "inline") || (bsonTag == "" && fieldType.Anonymous)
		if isInline {
			bsonTag = strings.TrimSuffix(bsonTag, ",inline")
		}

		// 如果bsonTag未定义且不是内联，则使用字段名
		if bsonTag == "" && !isInline {
			bsonTag = fieldType.Name
		}

		fullPath := prefix + bsonTag

		if freezeMap[fullPath] {
			continue
		}

		if sc.isSpecialStructType(fieldVal.Type()) {
			// 如果字段是特殊结构体类型，则直接返回值
			result[fullPath] = fieldVal.Interface()
			continue
		}

		if isInline || fieldVal.Kind() == reflect.Struct || fieldVal.Kind() == reflect.Map {
			inlinePrefix := prefix
			if !isInline {
				inlinePrefix += bsonTag + "."
			}
			inlineMap := sc.structToMap(fieldVal, freezeMap, inlinePrefix)
			for k, v := range inlineMap {
				result[k] = v
			}
			continue
		}

		result[fullPath] = fieldVal.Interface()
	}

	return result
}

// DiffToBsonM 缺点在于如果有内联的struct 如果原始数据中没有定义上层 则无法正确update
// 比如{city:{name:"新名称"}} 如果原始数据中没有 city字段则无法通过{city.name:"新名称"}进行更新?
func (sc *StructConverter) DiffToBsonM(original, current interface{}, freeze ...string) (bson.M, error) {
	originalVal := reflect.ValueOf(original)
	currentVal := reflect.ValueOf(current)

	if originalVal.Kind() == reflect.Ptr {
		originalVal = originalVal.Elem()
	}
	if currentVal.Kind() == reflect.Ptr {
		currentVal = currentVal.Elem()
	}

	if originalVal.Type() != currentVal.Type() {
		return nil, errors.New("输入必须是相同类型的结构体或结构体指针")
	}

	freezeMap := make(map[string]bool)
	for _, path := range freeze {
		freezeMap[path] = true
	}

	return sc.diffStructToMap(originalVal, currentVal, freezeMap, ""), nil
}

func (sc *StructConverter) diffStructToMap(originalVal, currentVal reflect.Value, freezeMap map[string]bool, prefix string) bson.M {
	result := bson.M{}
	typ := originalVal.Type()
	for i := 0; i < originalVal.NumField(); i++ {
		fieldType := typ.Field(i)
		bsonTag := fieldType.Tag.Get("bson")
		if bsonTag == "-" || !fieldType.IsExported() {
			continue
		}

		isInline := strings.Contains(bsonTag, "inline") || (bsonTag == "" && fieldType.Anonymous)
		if isInline {
			bsonTag = strings.TrimSuffix(bsonTag, ",inline")
		}

		// 如果bsonTag未定义且不是内联，则使用字段名
		if bsonTag == "" && !isInline {
			bsonTag = fieldType.Name
		}

		fullPath := bsonTag
		if prefix != "" && !isInline {
			fullPath = prefix + bsonTag
		}

		if freezeMap[fullPath] {
			continue
		}

		originalFieldVal := originalVal.Field(i)
		currentFieldVal := currentVal.Field(i)

		if isInline {
			// 处理带有inline标签的匿名字段
			inlineMap := sc.diffStructToMap(originalFieldVal, currentFieldVal, freezeMap, prefix)
			for k, v := range inlineMap {
				result[k] = v
			}
		} else if originalFieldVal.Kind() == reflect.Struct {
			// 处理嵌套结构体
			nestedPrefix := fullPath + "."
			nestedMap := sc.diffStructToMap(originalFieldVal, currentFieldVal, freezeMap, nestedPrefix)
			for k, v := range nestedMap {
				result[k] = v
			}
		} else if !cmp.Equal(originalFieldVal.Interface(), currentFieldVal.Interface()) {
			// 使用 cmp.Equal 比较字段值
			result[fullPath] = currentFieldVal.Interface()
		}
	}

	return result
}

// ToUpdateBson struct更新内容给转换为bson.M freeze支持格式 a.b.c
// 只会更新存在的 有值的 若原本有值要清除 请使用diff
func ToUpdateBson(input any, freeze ...string) (bson.M, error) {
	converter := NewStructConverter()
	converter.AddCustomIsZero(new(primitive.ObjectID), func(v reflect.Value, field reflect.StructField) bool {
		return v.Interface().(primitive.ObjectID).IsZero()
	})
	converter.AddCustomIsZero(new(time.Time), func(v reflect.Value, field reflect.StructField) bool {
		return v.Interface().(time.Time).IsZero()
	})
	return converter.StructToBsonM(input, freeze...)
}

// ToDiffBson 两个struct找出不同项 支持freeze格式为 a.b.c
// 只会对比值不同的 两个struct必须相同
func ToDiffBson[T any](original, current T, freeze ...string) (bson.M, error) {
	converter := NewStructConverter()

	return converter.DiffToBsonM(original, current, freeze...)
}

func NewStructConverter() *StructConverter {
	converter := &StructConverter{
		SpecialTypes: []reflect.Type{
			reflect.TypeOf(time.Time{}),
		},
	}
	converter.AddCustomIsZero(new(primitive.ObjectID), func(v reflect.Value, field reflect.StructField) bool {
		return v.Interface().(primitive.ObjectID).IsZero()
	})
	converter.AddCustomIsZero(new(time.Time), func(v reflect.Value, field reflect.StructField) bool {
		return v.Interface().(time.Time).IsZero()
	})
	return converter
}
