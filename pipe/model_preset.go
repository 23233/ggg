package pipe

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
		c.UpdateAt = time.Now()
	}
	if c.CreateAt.IsZero() {
		c.CreateAt = time.Now()
	}
	return nil
}
func (c *ModelBase) Reset() {
	c.Id = primitive.NewObjectID()
	c.Uid = SfNextId()
	c.UpdateAt = time.Now()
	c.CreateAt = time.Now()
}

func DefaultModelMap() map[string]any {
	var m = make(map[string]any, 0)
	m["_id"] = primitive.NewObjectID()
	m["uid"] = SfNextId()
	m["update_at"] = time.Now()
	m["create_at"] = time.Now()
	return m
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

type IMongoModel interface {
	IMongoBase
	GetCollName() string
	SyncIndex() error
}
