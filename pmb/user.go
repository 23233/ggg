package pmb

import (
	"github.com/23233/ggg/pipe"
	"strings"
)

const (
	UserModelName  = "users"
	UserContextKey = "user_model"
)

var (
	UserInstance    = NewUserModel()
	UserIdFieldName = "user_id"
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
	pipe.GenericsAccount `bson:",inline"`
	roleSecret           string
	connectInfo
}

func (c *SimpleUserModel) RoleSecret() string {
	if len(c.roleSecret) > 0 {
		return c.roleSecret
	}
	return "999888"
}
func (c *SimpleUserModel) SetRoleSecret(roleSecret string) {
	c.roleSecret = roleSecret
}

func (c *SimpleUserModel) GetUid() string {
	return c.Uid
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
