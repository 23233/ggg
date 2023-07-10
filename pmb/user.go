package pmb

import (
	"context"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

const (
	UserModelName = "users"
)

type UserModel interface {
	InjectDefault(mp map[string]any) error
	GetUid() string
}

type SimpleUserModel struct {
	Id        primitive.ObjectID `json:"_id" bson:"_id" `
	Uid       string             `json:"uid" bson:"uid"`
	UpdateAt  time.Time          `json:"update_at" bson:"update_at"`
	CreateAt  time.Time          `json:"create_at" bson:"create_at"`
	NickName  string             `json:"nick_name" bson:"nick_name,omitempty" comment:"昵称"`                 // 昵称
	AvatarUrl string             `json:"avatar_url" bson:"avatar_url,omitempty" comment:"头像地址" mab:"t=img"` // 头像地址
	UserName  string             `json:"-" bson:"user_name,omitempty" comment:"用户名"`
	Password  string             `json:"-" bson:"password,omitempty" comment:"用户登录密码"`
	Salt      string             `json:"-" bson:"salt,omitempty" comment:"密码加密salt"`
	TelPhone  string             `json:"tel_phone,omitempty" bson:"tel_phone,omitempty" comment:"电话号码"` // 电话号码
	Balance   uint64             `json:"balance,omitempty" bson:"balance,omitempty" comment:"余额(分)" `   // 余额 单位是分
}

func (c *SimpleUserModel) InjectDefault(mp map[string]any) error {
	v, ok := mp["_id"]
	if ok {
		c.Id = v.(primitive.ObjectID)
	}
	v, ok = mp["uid"]
	if ok {
		c.Uid = v.(string)
	}
	v, ok = mp["update_at"]
	if ok {
		c.UpdateAt = v.(time.Time)
	}
	v, ok = mp["create_at"]
	if ok {
		c.CreateAt = v.(time.Time)
	}
	return nil
}
func (c *SimpleUserModel) GetUid() string {
	return c.Uid
}
func (c *SimpleUserModel) BeforeInsert(ctx context.Context) error {
	if c.Id.IsZero() {
		return errors.New("未注入默认数据")
	}
	return nil
}
