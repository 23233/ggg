package ut

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"strings"
	"time"
)

func getTagName(tag string) string {
	parts := strings.Split(tag, ",")
	return parts[0]
}

// findFieldByJSONTag recursively searches for a struct field with the specified JSON tag.
func findFieldByJSONTag(v reflect.Value, tag string, handleField func(reflect.Value, reflect.StructField)) (reflect.Value, bool) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	typ := v.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := v.Field(i)
		fieldJson := getTagName(field.Tag.Get("json"))
		if len(fieldJson) < 1 || fieldJson == "-" {
			// Handle embedded struct without JSON tag
			if field.Anonymous && fieldValue.Kind() == reflect.Struct {
				if field, found := findFieldByJSONTag(fieldValue, tag, handleField); found {
					return field, true
				}
			}
			continue
		}
		if fieldJson == tag {
			if handleField != nil {
				handleField(fieldValue, field)
			}
			return fieldValue, true
		}
		if fieldValue.Kind() == reflect.Struct {
			if field, found := findFieldByJSONTag(fieldValue, tag, handleField); found {
				return field, true
			}
		}
	}
	return reflect.Value{}, false
}
func setFieldValue(field reflect.Value, value interface{}) error {
	switch field.Type() {
	case reflect.TypeOf(primitive.ObjectID{}):
		if strValue, ok := value.(string); ok {
			oid, err := primitive.ObjectIDFromHex(strValue)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(oid))
		} else {
			return fmt.Errorf("cannot convert %v to ObjectID", value)
		}
	case reflect.TypeOf(time.Time{}):
		if strValue, ok := value.(string); ok {
			t, err := time.Parse(time.RFC3339, strValue)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(t))
		} else {
			return fmt.Errorf("cannot convert %v to Time", value)
		}
	default:
		valueV := reflect.ValueOf(value)
		if valueV.Type() != field.Type() {
			return fmt.Errorf("type mismatch: cannot assign %v to field with type %v", valueV.Type(), field.Type())
		}
		field.Set(valueV)
	}
	return nil
}

// ModelCtxMapperPack 解析body数据包 所有map key 支持 a.b.c 但是对应的是json tag标签名
type ModelCtxMapperPack struct {
	InjectData map[string]any `json:"inject_data,omitempty"` // 注入数据 默认覆盖
	DropKeys   []string       `json:"drop_keys,omitempty"`   // 需要丢弃的key
}

func (m *ModelCtxMapperPack) processStruct(data any) error {
	v := reflect.ValueOf(data).Elem()

	// Drop keys.
	for _, key := range m.DropKeys {
		field, found := findFieldByJSONTag(v, key, nil)
		if !found {
			continue
		}
		field.Set(reflect.Zero(field.Type()))
	}

	// Inject data.
	for key, value := range m.InjectData {
		field, found := findFieldByJSONTag(v, key, nil)
		if !found {
			return fmt.Errorf("field with JSON tag %q not found", key)
		}
		if err := setFieldValue(field, value); err != nil {
			return err
		}
	}

	return nil
}

func (m *ModelCtxMapperPack) processMap(data map[string]any) error {
	for _, key := range m.DropKeys {
		delete(data, key)
	}

	for key, val := range m.InjectData {
		data[key] = val
	}
	return nil
}

func (m *ModelCtxMapperPack) Process(data any) error {
	switch v := data.(type) {
	case map[string]any:
		return m.processMap(v)
	default:
		return m.processStruct(data)
	}

}
