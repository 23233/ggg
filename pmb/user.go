package pmb

import (
	"context"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strings"
	"time"
)

const (
	UserModelName  = "users"
	UserContextKey = "user_model"
)

var (
	UserInstance = NewUserModel()
)

type UserModel interface {
	InjectDefault(mp map[string]any) error
	GetUid() string
}

type Platform struct {
	Name      string `json:"name,omitempty" bson:"name,omitempty" comment:"平台名称"`
	Id        string `json:"id,omitempty" bson:"id,omitempty" comment:"平台唯一ID"`
	NickName  string `json:"nick_name,omitempty" bson:"nick_name,omitempty" comment:"平台昵称"`
	AvatarUrl string `json:"avatar_url,omitempty" bson:"avatar_url,omitempty" comment:"平台头像"`
	Data      string `json:"data,omitempty" bson:"data,omitempty" comment:"平台数据"`
}

type SimpleUserModel struct {
	Id        primitive.ObjectID `json:"_id" bson:"_id"`
	Uid       string             `json:"uid" bson:"uid"`
	UpdateAt  time.Time          `json:"update_at" bson:"update_at"`
	CreateAt  time.Time          `json:"create_at" bson:"create_at"`
	NickName  string             `json:"nick_name,omitempty" bson:"nick_name,omitempty" comment:"昵称"`                 // 昵称
	AvatarUrl string             `json:"avatar_url,omitempty" bson:"avatar_url,omitempty" comment:"头像地址" mab:"t=img"` // 头像地址
	UserName  string             `json:"user_name,omitempty" bson:"user_name,omitempty" comment:"用户名"`
	Password  string             `json:"password,omitempty" bson:"password,omitempty" comment:"用户登录密码"`
	Salt      string             `json:"salt,omitempty" bson:"salt,omitempty" comment:"密码加密salt"`
	TelPhone  string             `json:"tel_phone,omitempty" bson:"tel_phone,omitempty" comment:"电话号码"` // 电话号码
	Balance   uint64             `json:"balance,omitempty" bson:"balance,omitempty" comment:"余额(分)" `   // 余额 单位是分
	Email     string             `json:"email,omitempty" bson:"email,omitempty" comment:"邮箱地址"`
	Platforms []Platform         `json:"platforms,omitempty" bson:"platforms,omitempty" comment:"平台信息"`
	connectInfo
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

// Masking 数据脱敏 传入level 1隐藏号码 2隐藏余额
func (c *SimpleUserModel) Masking(level int) *SimpleUserModel {
	var user = new(SimpleUserModel)
	*user = *c
	user.UserName = ""
	user.Password = ""
	user.Salt = ""

	if level >= 1 {
		// 若号码长度小于2就错误了
		if len(c.TelPhone) > 2 {
			user.TelPhone = user.TelPhone[0:1] + strings.Repeat("*", len(user.TelPhone)-2) + user.TelPhone[len(user.TelPhone)-1:]
		}
	}

	switch level {
	case 2:
		user.Balance = 0
	}
	return user
}

func NewUserModel(conn ...connectInfo) *SimpleUserModel {
	um := new(SimpleUserModel)
	if len(conn) >= 1 {
		um.connectInfo = conn[0]
	}
	return um
}
