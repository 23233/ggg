package pmb

import (
	"context"
	"github.com/23233/ggg/pipe"
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
)

type UserPasswordLoginReq struct {
	UserName string `json:"user_name,omitempty" comment:"用户名" validate:"required,min=3,max=24"`
	Password string `json:"password,omitempty" comment:"密码" validate:"required,min=6,max=36"`
	Force    bool   `json:"force,omitempty" comment:"强制" `
}

func (c *SimpleUserModel) SyncIndex(ctx context.Context) error {
	cl, err := c.db.Collection(UserModelName).CloneCollection()
	if err != nil {
		return err
	}
	err = ut.MCreateIndex(ctx, cl, ut.MGenUnique("uid", false), ut.MGenUnique("user_name", true), ut.MGenUnique("tel_phone", true))
	return err
}
func (c *SimpleUserModel) LoginUsePasswordHandler() iris.Handler {
	return func(ctx iris.Context) {
		var body = new(UserPasswordLoginReq)
		err := ctx.ReadBody(&body)
		if err != nil {
			IrisRespErr("解析请求包参数错误", err, ctx)
			return
		}

		rateResp := pipe.RequestRate.Run(ctx, nil, &pipe.RateLimitPipe{
			RatePeriod: "5-M",
			KeyGen: &pipe.StrExpand{
				Key: "lf:${un}",
				KeyMap: []pipe.StrTemplate{
					{
						VarName: "un",
						Value:   body.UserName,
					},
				},
			},
			WriteHeader: false,
		}, nil)
		if rateResp.Err != nil {
			IrisRespErr("操作过快", rateResp.Err, ctx, rateResp.ReqCode)
			return
		}

		userModel, err := c.GetUserItem(ctx, bson.M{"user_name": body.UserName})
		if err != nil {
			IrisRespErr("用户名或密码有误", err, ctx)
			return
		}

		// 验证密码是否正确
		if !validPassword(body.Password, userModel.Salt, userModel.Password) {
			IrisRespErr("用户名或密码错误", nil, ctx)
			return
		}

		// 正确的情况下 判断是否可以登录
		disableLoginResp := pipe.RbacAllow.Run(ctx, nil, &pipe.RbacAllowPipe{
			Sub:    userModel.Uid,
			Obj:    pipe.RbacNotAllowLoginObj,
			Domain: pipe.RbacSelfDomainName,
			Act:    pipe.RbacNormalAct,
		}, c.rbac)

		// 在这个组里就是不允许登录的
		if disableLoginResp.Result {
			IrisRespErr("该用户被禁止登录", nil, ctx)
			return
		}

		userModel.connectInfo = c.connectInfo
		token, err := userModel.GenJwtToken(ctx, body.Force)
		if err != nil {
			IrisRespErr("生成登录令牌失败", err, ctx)
			return
		}

		ctx.JSON(iris.Map{"token": token, "info": userModel.Masking(0)})
	}
}
func (c *SimpleUserModel) RegistryUseUserNamePassword() iris.Handler {
	return func(ctx iris.Context) {
		var body = new(UserPasswordLoginReq)
		err := ctx.ReadBody(&body)
		if err != nil {
			IrisRespErr("解析请求包参数错误", err, ctx)
			return
		}

		userModel, err := c.GetUserItem(ctx, bson.M{"user_name": body.UserName})
		if err != nil {
			if err != qmgo.ErrNoSuchDocuments {
				IrisRespErr("用户名已存在", err, ctx)
				return
			}
		}
		if userModel != nil && !userModel.Id.IsZero() {
			IrisRespErr("用户已存在", err, ctx)
			return
		}
		userModel = new(SimpleUserModel)
		// 进行注册
		password, salt := passwordSalt(body.Password)
		userModel.UserName = body.UserName
		userModel.Password = password
		userModel.Salt = salt
		userModel.NickName = ut.RandomStr(12)
		err = userModel.InjectDefault(pipe.DefaultModelMap())
		if err != nil {
			IrisRespErr("写入默认信息失败", err, ctx, 500)
			return
		}
		// 插入用户
		_, err = c.db.Collection(UserModelName).InsertOne(ctx, &userModel)
		if err != nil {
			IrisRespErr("新增用户失败", err, ctx, 500)
			return
		}
		userModel.connectInfo = c.connectInfo
		token, err := userModel.GenJwtToken(ctx, false)
		if err != nil {
			IrisRespErr("生成登录令牌失败", err, ctx)
			return
		}
		ctx.JSON(iris.Map{"token": token, "info": userModel.Masking(0)})
	}
}
func (c *SimpleUserModel) MustLoginMiddleware() iris.Handler {
	return func(ctx iris.Context) {
		// 进行jwt验证
		resp := pipe.JwtVisit.Run(ctx, &pipe.JwtCheckDep{
			Env:           pipe.CtxGetEnv(ctx),
			Authorization: ctx.GetHeader("Authorization"),
		}, nil, c.rdb)
		if resp.Err != nil {
			IrisRespErr("验证登录状态失败", resp.Err, ctx, http.StatusUnauthorized)
			ctx.StopExecution()
			return
		}

		// 根据用户id 获取到用户信息
		userModel, err := c.GetUserItem(ctx, bson.M{"uid": resp.Result.UserId})
		if err != nil {
			IrisRespErr("获取出用户信息失败", err, ctx)
			return
		}

		ctx.Values().Set("_jwt", resp.Result)
		ctx.Values().Set(UserContextKey, userModel)
		ctx.Next()
	}
}
func (c *SimpleUserModel) GetUserItem(ctx context.Context, filter bson.M) (*SimpleUserModel, error) {
	// 判断用户是否存在
	userModel := new(SimpleUserModel)
	err := c.db.Collection(UserModelName).Find(ctx, filter).One(&userModel)
	if err != nil {
		return nil, err
	}
	userModel.connectInfo = c.connectInfo
	return userModel, nil
}
func (c *SimpleUserModel) GenJwtToken(ctx iris.Context, force bool) (string, error) {
	// 生成token
	jwtResp := pipe.JwtGen.Run(ctx, &pipe.PipeJwtDep{
		Env:    pipe.CtxGetEnv(ctx),
		UserId: c.Uid,
	}, &pipe.JwtGenPipe{
		Force: force,
	}, c.rdb)
	return jwtResp.Result, jwtResp.Err
}

func (c *SimpleUserModel) SetRoleUseUserName(ctx context.Context, userName string, roleTarget string) error {
	var err error
	user, err := c.GetUserItem(ctx, bson.M{"user_name": userName})
	if err != nil {
		return err
	}
	switch roleTarget {
	case "root":
		_, err = c.rbac.SetRoot(user.Uid)
	default:
		_, err = c.rbac.SetStaff(user.Uid)
	}
	return err
}
func (c *SimpleUserModel) RemoveUser(ctx context.Context, uid string) error {
	return c.db.Collection(UserModelName).Remove(ctx, bson.M{ut.DefaultUidTag: uid})
}
