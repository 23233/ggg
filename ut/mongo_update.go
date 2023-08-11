package ut

import (
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"reflect"
	"strings"
)

// StructConverter 作用在于struct中如果需要更新的时候 不要使用bson.M来手动设置字段名
// 这样又没有编辑器提示 字段变更时也很难同步 可以采取这种方式
// 特点是会自动去除0值的字段最后只保留有值的产出bson.M
// 那在某些情况下 会从db中取出存储的哪一行 然后跟当前的进行对比
// 这时候就使用 diff 的模式 对比两个struct中的不同项进行变更
type StructConverter struct {
	CustomIsZero map[reflect.Type]func(reflect.Value, reflect.StructField) bool
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

		if sc.IsZero(fieldVal, fieldType) {
			continue
		}

		bsonTag := fieldType.Tag.Get("bson")
		if bsonTag == "-" {
			continue
		}

		bsonTagParts := strings.Split(bsonTag, ",")
		bsonTag = bsonTagParts[0]

		isInline := len(bsonTagParts) > 1 && bsonTagParts[1] == "inline"
		isAnonymous := fieldType.Anonymous

		fullPath := prefix + bsonTag

		if freezeMap[fullPath] {
			continue
		}

		if isInline || isAnonymous {
			if fieldVal.Kind() == reflect.Struct {
				inlinePrefix := fullPath
				if isAnonymous {
					inlinePrefix = prefix // 对于匿名字段，我们不添加额外的前缀
				}
				inlineMap := sc.structToMap(fieldVal, freezeMap, inlinePrefix+".")
				for k, v := range inlineMap {
					result[k] = v
				}
				continue
			}
		}

		if fieldVal.Kind() == reflect.Struct {
			nestedMap := sc.structToMap(fieldVal, freezeMap, fullPath+".")
			if len(nestedMap) > 0 {
				result[bsonTag] = nestedMap
			}
		} else {
			result[bsonTag] = fieldVal.Interface()
		}
	}

	return result
}

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
		if bsonTag == "-" {
			continue
		}

		bsonTagParts := strings.Split(bsonTag, ",")
		bsonTag = bsonTagParts[0]

		isInline := len(bsonTagParts) > 1 && bsonTagParts[1] == "inline" || fieldType.Type.Kind() == reflect.Struct

		fullPath := prefix + bsonTag

		if freezeMap[fullPath] {
			continue
		}

		originalFieldVal := originalVal.Field(i)
		currentFieldVal := currentVal.Field(i)

		if isInline {
			if originalFieldVal.Kind() == reflect.Struct && currentFieldVal.Kind() == reflect.Struct {
				inlineMap := sc.diffStructToMap(originalFieldVal, currentFieldVal, freezeMap, fullPath+".")
				for k, v := range inlineMap {
					result[k] = v
				}
				continue
			}
		}

		if !cmp.Equal(originalFieldVal.Interface(), currentFieldVal.Interface()) {
			if currentFieldVal.Kind() == reflect.Struct {
				result[bsonTag] = sc.diffStructToMap(originalFieldVal, currentFieldVal, freezeMap, fullPath+".")
			} else {
				result[bsonTag] = currentFieldVal.Interface()
			}
		}
	}

	return result
}

// ToUpdateBson struct更新内容给转换为bson.M freeze支持格式 a.b.c
// 只会更新存在的 有值的 若原本有值要清除 请使用diff
func ToUpdateBson(input any, freeze ...string) (bson.M, error) {
	converter := new(StructConverter)
	return converter.StructToBsonM(input, freeze...)
}

// ToDiffBson 两个struct找出不同项 支持freeze格式为 a.b.c
// 只会对比值不同的 两个struct必须相同
func ToDiffBson[T any](original, current T, freeze ...string) (bson.M, error) {
	converter := new(StructConverter)
	return converter.DiffToBsonM(original, current, freeze...)
}
