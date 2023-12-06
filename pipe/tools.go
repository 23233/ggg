package pipe

import (
	"github.com/23233/jsonschema"
	uuid "github.com/iris-contrib/go.uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"strings"
	"time"
)

func GenUUid() string {
	uidV4, _ := uuid.NewV4()
	OutTradeNo := strings.ReplaceAll(uidV4.String(), "-", "")
	return OutTradeNo
}

func ToJsonSchema[T any](origin T, omitFields ...string) *jsonschema.Schema {
	schema := new(jsonschema.Reflector)
	// 只要为标记为omitempty的都会进入required
	//schema.RequiredFromJSONSchemaTags = true
	// 用真实的[]uint8 别去mock去一个 string base64出来
	schema.DoNotBase64 = true
	// 为true 则会写入Properties 对于object会写入$defs 生成$ref引用
	schema.ExpandedStruct = true
	schema.Mapper = func(r reflect.Type) *jsonschema.Schema {
		switch r {
		case reflect.TypeOf(primitive.ObjectID{}):
			return &jsonschema.Schema{
				Type:   "string",
				Format: "objectId",
			}
		case reflect.TypeOf(time.Time{}):
			return &jsonschema.Schema{
				Type:   "string",
				Format: "date-time",
			}
		}

		return nil
	}

	// 映射comment为Title
	schema.AddTagSetMapper("comment", "Title")

	// 还应该跳过_id 和uid 不让改
	schema.Intercept = func(field reflect.StructField) bool {
		if field.Name == "Id" && field.Type == reflect.TypeOf(primitive.ObjectID{}) {
			return false
		}
		if field.Name == "Uid" && field.Type == reflect.TypeOf("") {
			return false
		}
		for _, omitField := range omitFields {
			if field.Name == omitField {
				return false
			}
		}
		return true
	}

	ref := schema.Reflect(origin)
	return ref
}
