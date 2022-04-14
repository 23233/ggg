package smab

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io/ioutil"
	"time"
)

var (
	defaultRootName      = "root"
	userModelContextName = "u"
)

var (
	RootUser *SmUserModel
)

type DefaultField struct {
	Id       primitive.ObjectID `bson:"_id" json:"id" comment:"id"`
	UpdateAt time.Time          `bson:"update_at" json:"update_at" comment:"更新时间"`
	CreateAt time.Time          `json:"create_at" bson:"create_at" comment:"创建时间"`
}

func (u *DefaultField) BeforeInsert(ctx context.Context) error {
	if u.Id.IsZero() {
		u.Id = primitive.NewObjectID()
	}
	u.UpdateAt = time.Now().Local()
	u.CreateAt = time.Now().Local()
	return nil
}

func (u *DefaultField) BeforeUpdate(ctx context.Context) error {
	u.UpdateAt = time.Now().Local()
	return nil
}

func (u *DefaultField) BeforeUpsert(ctx context.Context) error {
	u.UpdateAt = time.Now().Local()
	return nil
}

type QianKunConfigExtra struct {
	Name  string `json:"name" bson:"name" comment:"子应用名称"`
	Entry string `json:"entry" bson:"entry" comment:"子应用 html 地址"`
	Path  string `json:"path" bson:"path" comment:"子应用访问路径"`
	Label string `json:"label" bson:"label" comment:"子应用中文名"`
}

type FilterDataExtra struct {
	ModelName string   `json:"model_name" bson:"model_name" comment:"模型名称"`
	Key       []string `json:"key" bson:"key" comment:"键"`
	Value     string   `json:"value" bson:"value" comment:"值"`
}

// SmUserModel 管理后台用户
type SmUserModel struct {
	DefaultField `bson:",inline,flatten" `
	Name         string               `json:"name" bson:"name" comment:"用户名"`
	Password     string               `json:"password" bson:"password" comment:"加密密码"`
	Salt         string               `json:"salt" bson:"salt" comment:"salt"`
	Desc         string               `json:"desc" bson:"desc" comment:"描述"`
	Phone        string               `json:"phone" bson:"phone" comment:"手机号"`
	SuperUser    bool                 `json:"super_user" bson:"super_user" comment:"是否超级用户?"`
	CreateId     primitive.ObjectID   `json:"create_id" bson:"create_id" comment:"创建者ID"` // 创建者ID
	QianKun      []QianKunConfigExtra `json:"qian_kun,omitempty" bson:"qian_kun,omitempty" comment:"乾坤配置"`
	FilterData   []FilterDataExtra    `json:"filter_data" bson:"filter_data" comment:"过滤数据"`
}

// 用户是否为超级管理员
func (u *SmUserModel) isSuper() bool {
	return u.Name == defaultRootName || u.SuperUser
}

func (u *SmUserModel) isRoot() bool {
	return u.Name == defaultRootName
}

func (u *SmUserModel) getIdStr() string {
	return u.Id.Hex()
}

// 密码加密
func passwordSalt(plaintext string) (string, string) {
	salt := randomStr(4)
	m5 := md5.New()
	m5.Write([]byte(plaintext))
	m5.Write([]byte(salt))
	st := m5.Sum(nil)
	ps := hex.EncodeToString(st)
	return ps, salt
}

// 密码验证
func passwordValid(plaintext, salt, password string) bool {
	r := md5.New()
	r.Write([]byte(plaintext))
	r.Write([]byte(salt))
	st := r.Sum(nil)
	ps := hex.EncodeToString(st)
	return ps == password
}

// IdGetUser id获取用户
func IdGetUser(ctx context.Context, id string) (*SmUserModel, error) {
	obj, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	return AnyGetUser(ctx, bson.M{"_id": obj})
}

// NameGetUser name获取用户
func NameGetUser(ctx context.Context, name string) (*SmUserModel, error) {
	return AnyGetUser(ctx, bson.M{"name": name})
}

// AnyGetUser 任意参数获取用户
func AnyGetUser(ctx context.Context, params bson.M) (*SmUserModel, error) {
	var u SmUserModel
	err := getCollName("sm_user_model").Find(ctx, params).One(&u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateUser 新增用户
func CreateUser(ctx context.Context, u *SmUserModel) (*SmUserModel, error) {
	one, err := getCollName("sm_user_model").InsertOne(ctx, u)
	if err != nil {
		return nil, err
	}
	if one.InsertedID.(primitive.ObjectID).IsZero() {
		return nil, errors.New("新增失败")
	}
	return u, nil
}

// 用户id获取用户中间件
func idGetUserMiddleware(ctx iris.Context) {
	id := ctx.Values().Get("uid").(string)
	if len(id) >= 1 {
		u, err := IdGetUser(ctx.Request().Context(), id)
		if err == nil {
			_ = ctx.SetUser(u)
			ctx.Values().Set(userModelContextName, u)
			ctx.Next()
			return
		}
	}
	ctx.StatusCode(iris.StatusUnauthorized)
	_, _ = ctx.JSON(iris.Map{"detail": "获取当前用户失败,请重新登录"})
	ctx.StopExecution()
	return

}

// 初始化管理员
func (lib *SpAdmin) initSuperUser() bool {
	// 判断用户是否存在

	u, err := NameGetUser(context.Background(), defaultRootName)
	if err != nil {
		if err != qmgo.ErrNoSuchDocuments {
			return false
		}
	}
	if u == nil || u.Id.IsZero() {
		if u == nil {
			u = new(SmUserModel)
		}
		u.Name = defaultRootName
		password := randomStr(12)
		pwd, salt := passwordSalt(password)
		u.Password = pwd
		u.Salt = salt
		u.SuperUser = true
		one, err := getCollName("sm_user_model").InsertOne(context.Background(), u)
		if err != nil || one.InsertedID.(primitive.ObjectID).IsZero() {
			fmt.Println("创建后台超级用户失败")
			return false
		}
		fmt.Println(fmt.Sprintf("超级用户密码为:%s", password))

		// 写入到本地文件中去
		if err = ioutil.WriteFile("./admin_init_password.txt", []byte(password), 0666); err != nil {
			fmt.Println("写入密码到本地文件失败")
		}

		fmt.Println(fmt.Sprintf("超级用户Id为:%s", u.Id.Hex()))
		RootUser = u
		return true
	} else {
		if lib.config.AllowTokenLogin {
			// 生成重置密钥
			lib.rootLoginToken = randomStr(32)
			fmt.Println(fmt.Sprintf("root重置登录密钥为:%s", lib.rootLoginToken))
		}
	}
	RootUser = u
	return false

}
