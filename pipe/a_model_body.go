package pipe

import (
	"fmt"
	"github.com/kataras/iris/v12"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"time"
)

// findFieldByJSONTag recursively searches for a struct field with the specified JSON tag.
func findFieldByJSONTag(v reflect.Value, tag string) (reflect.Value, bool) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	typ := v.Type()
	for i := 0; i < typ.NumField(); i++ {
		if typ.Field(i).Tag.Get("json") == tag {
			return v.Field(i), true
		}
		if v.Field(i).Kind() == reflect.Struct {
			if field, found := findFieldByJSONTag(v.Field(i), tag); found {
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
	InjectData map[string]any        `json:"inject_data,omitempty"`                        // 注入数据 默认覆盖
	DropKeys   []string              `json:"drop_keys,omitempty"`                          // 需要丢弃的key
	GenKeys    map[string]*Attribute `json:"gen_keys,omitempty" bson:"gen_keys,omitempty"` // 代码生成的数据
}

func (m *ModelCtxMapperPack) processStruct(data any) error {
	v := reflect.ValueOf(data).Elem()

	// Drop keys.
	for _, key := range m.DropKeys {
		field, found := findFieldByJSONTag(v, key)
		if !found {
			continue
		}
		field.Set(reflect.Zero(field.Type()))
	}

	// Generate keys.
	for key, gen := range m.GenKeys {
		field, found := findFieldByJSONTag(v, key)
		if !found {
			return fmt.Errorf("field with JSON tag %q not found", key)
		}
		genValue, err := gen.RunGen()
		if err != nil {
			return fmt.Errorf("error generating value for key %q: %v", key, err)
		}
		if err := setFieldValue(field, genValue); err != nil {
			return err
		}
	}

	// Inject data.
	for key, value := range m.InjectData {
		field, found := findFieldByJSONTag(v, key)
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
	for key, attr := range m.GenKeys {
		result, err := attr.RunGen()
		if err != nil {
			return err
		}
		data[key] = result
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

var (
	ModelMapper = &RunnerContext[any, *ModelCtxMapperPack, any, any]{
		Key:  "model_ctx_mapper",
		Name: "模型body取map",
		call: func(ctx iris.Context, origin any, params *ModelCtxMapperPack, db any, more ...any) *PipeRunResp[any] {
			var bodyData = origin
			if origin == nil {
				bodyData = make(map[string]any)
			}
			err := ctx.ReadBody(&bodyData)
			if err != nil {
				return newPipeErr[any](err)
			}

			err = params.Process(bodyData)
			if err != nil {
				return newPipeErr[any](err)
			}

			return newPipeResult(bodyData)

		},
	}
)
