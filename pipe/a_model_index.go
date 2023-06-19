package pipe

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ModelIndex struct {
	FieldName []string `json:"field_name,omitempty" comment:"字段名"` // 支持多级 按.分割 数组是因为可以支持联合索引 这取决于保存的后端是否支持
	Type      string   `json:"type,omitempty" comment:"类型"`        // 2dSphere text unique unique_sparse normal
}

func (c *ModelIndex) Gen(v any) mongo.IndexModel {
	var m = bson.M{}
	for _, s := range c.FieldName {
		m[s] = v
	}
	return mongo.IndexModel{
		Keys: m,
	}
}

func (c *ModelIndex) NewModelTextIndex(fields ...string) *ModelIndex {
	return &ModelIndex{FieldName: fields, Type: "text"}
}
func (c *ModelIndex) NewModelNormalIndex(fields ...string) *ModelIndex {
	return &ModelIndex{FieldName: fields, Type: "normal"}
}
func (c *ModelIndex) NewModelUniqueIndex(fields ...string) *ModelIndex {
	return &ModelIndex{FieldName: fields, Type: "unique"}
}
func (c *ModelIndex) NewModelUniqueSparseIndex(fields ...string) *ModelIndex {
	return &ModelIndex{FieldName: fields, Type: "unique_sparse"}
}

func (c *ModelIndex) NewModel2dSphereIndex(fields ...string) *ModelIndex {
	return &ModelIndex{FieldName: fields, Type: "2dSphere"}
}
