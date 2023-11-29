package pipe

import (
	"context"
	jsoniter "github.com/json-iterator/go"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"time"
)

type ModelBase struct {
	Id       primitive.ObjectID `json:"_id,omitempty" bson:"_id"`
	Uid      string             `json:"uid,omitempty" bson:"uid,omitempty"`
	UpdateAt time.Time          `json:"update_at,omitempty" bson:"update_at,omitempty" comment:"更新时间"`
	CreateAt time.Time          `json:"create_at,omitempty" bson:"create_at,omitempty" comment:"创建时间"`
}

func (c *ModelBase) BeforeInsert(ctx context.Context) error {
	if c.Id.IsZero() {
		c.Id = primitive.NewObjectID()
	}
	if len(c.Uid) < 1 {
		c.Uid = SfNextId()
	}
	if c.UpdateAt.IsZero() {
		c.UpdateAt = time.Now().Local()
	}
	if c.CreateAt.IsZero() {
		c.CreateAt = time.Now().Local()
	}
	return nil
}
func (c *ModelBase) Reset() {
	c.Id = primitive.NewObjectID()
	c.Uid = SfNextId()
	c.UpdateAt = time.Now().Local()
	c.CreateAt = time.Now().Local()
}

func DefaultModelMap() map[string]any {
	var m = make(map[string]any, 0)
	m["_id"] = primitive.NewObjectID()
	m["uid"] = SfNextId()
	m["update_at"] = time.Now().Local()
	m["create_at"] = time.Now().Local()
	return m
}

func StructToMap(s any) (map[string]any, error) {
	data, err := jsoniter.Marshal(s)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	err = jsoniter.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	// 遍历map，删除值为nil或空map的键值对
	for k, v := range m {
		if v == nil {
			delete(m, k)
		} else if reflect.TypeOf(v).Kind() == reflect.Map {
			subMap, ok := v.(map[string]interface{})
			if ok && len(subMap) == 0 {
				delete(m, k)
			}
		}
	}

	return m, nil
}

func (c *ModelBase) GetBase() *ModelBase {
	return c
}
func (c *ModelBase) SetBase(raw ModelBase) {
	c.Uid = raw.Uid
	c.Id = raw.Id
	c.UpdateAt = raw.UpdateAt
	c.CreateAt = raw.CreateAt
}

type IMongoBase interface {
	GetBase() *ModelBase
	SetBase(raw ModelBase)
}
