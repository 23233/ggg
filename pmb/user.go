package pmb

import (
	"github.com/23233/ggg/pipe"
	"github.com/kataras/iris/v12"
	"strings"
)

const (
	UserModelName    = "users"
	UserContextKey   = "user_model"
	UserIdContextKey = "user_model_id"
)

var (
	UserInstance    = NewUserModel()
	UserIdFieldName = "user_id"
	UserCookieKey   = "tk"
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

type SimpleUserHooks struct {
	OnLoginAfter func(ctx iris.Context, user *SimpleUserModel, token string) error
}

type SimpleUserModel struct {
	pipe.GenericsAccount `bson:",inline"`
	hooks                *SimpleUserHooks
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

func (c *SimpleUserModel) SetHooks(hooks *SimpleUserHooks) {
	c.hooks = hooks
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
	user.LastUa = ""
	user.RegUa = ""
	user.RegIp = ""
	user.LastIp = ""

	if level >= 1 {
		// 若号码长度小于2就错误了
		if len(c.TelPhone) > 2 {
			user.TelPhone = user.TelPhone[0:1] + strings.Repeat("*", len(user.TelPhone)-2) + user.TelPhone[len(user.TelPhone)-1:]
		}
		if len(c.Email) > 1 {
			user.Email = ""
		}
	}

	switch level {
	case 2:
		user.Balance = 0
	}
	return user
}

func (c *SimpleUserModel) OpenIdGetPlatform(openid string) *pipe.AccountPlatform {
	if c.Platforms != nil {
		for _, v := range c.Platforms {
			if v.Pid == openid {
				return v
			}
		}
	}
	return nil
}

func (c *SimpleUserModel) PlatformIsExist(name string) bool {
	if c.Platforms != nil {
		for _, v := range c.Platforms {
			if v.Name == name {
				return true
			}
		}
	}
	return false
}

func (c *SimpleUserModel) AppidGetPlatform(appid string) *pipe.AccountPlatform {
	if c.Platforms != nil {
		for _, v := range c.Platforms {
			if v.Appid == appid {
				return v
			}
		}
	}
	return nil
}

func NewUserModel(conn ...connectInfo) *SimpleUserModel {
	um := new(SimpleUserModel)
	if len(conn) >= 1 {
		um.connectInfo = conn[0]
	}
	return um
}
