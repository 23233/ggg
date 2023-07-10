package pmb

import (
	"context"
	"github.com/23233/ggg/pipe"
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/qiniu/qmgo"
	"github.com/redis/rueidis"
	"go.mongodb.org/mongo-driver/bson"
)

type UserPasswordLoginReq struct {
	UserName string `json:"user_name,omitempty" comment:"用户名" validate:"required,min=3,max=24"`
	Password string `json:"password,omitempty" comment:"密码" validate:"required,min=6,max=36"`
	Force    bool   `json:"force,omitempty" comment:"强制" `
}

func (c *SimpleUserModel) SyncIndex(ctx context.Context, db *qmgo.Database) error {
	cl, err := db.Collection(UserModelName).CloneCollection()
	if err != nil {
		return err
	}
	err = ut.MCreateIndex(ctx, cl, ut.MGenUnique("uid", false), ut.MGenUnique("user_name", true), ut.MGenUnique("tel_phone", true))
	return err
}

func (c *SimpleUserModel) LoginUsePasswordHandler(db *qmgo.Database, rdb rueidis.Client, domain *pipe.RbacDomain) iris.Handler {
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
		// 判断用户是否存在
		userModel := new(SimpleUserModel)
		err = db.Collection(UserModelName).Find(ctx, bson.M{"user_name": body.UserName}).One(&userModel)
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
			Sub:       userModel.Uid,
			Obj:       pipe.RbacAllowLoginObj,
			Domain:    pipe.RbacSelfDomainName,
			Act:       pipe.RbacNormalAct,
			OnlyCheck: true,
		}, domain)
		// 在这个组里就是不允许登录的
		if disableLoginResp.Err != nil || disableLoginResp.Result {
			IrisRespErr("该用户被禁止登录", nil, ctx)
			return
		}

		// 生成token
		jwtResp := pipe.JwtGen.Run(ctx, &pipe.PipeJwtDep{
			Env:    pipe.CtxGetEnv(ctx),
			UserId: userModel.Uid,
		}, &pipe.JwtGenPipe{
			Force: body.Force,
		}, rdb)

		if jwtResp.Err != nil {
			IrisRespErr("生成登录令牌失败", jwtResp.Err, ctx, jwtResp.ReqCode)
			return
		}

		ctx.JSON(iris.Map{"token": jwtResp.Result})
	}
}

func (c *SimpleUserModel) RegistryUseUserNamePassword(db *qmgo.Database, rdb rueidis.Client) iris.Handler {
	return func(ctx iris.Context) {
		var body = new(UserPasswordLoginReq)
		err := ctx.ReadBody(&body)
		if err != nil {
			IrisRespErr("解析请求包参数错误", err, ctx)
			return
		}
		// 判断用户是否存在
		userModel := new(SimpleUserModel)
		err = db.Collection(UserModelName).Find(ctx, bson.M{"user_name": body.UserName}).One(&userModel)
		if err != nil {
			if err != qmgo.ErrNoSuchDocuments {
				IrisRespErr("用户名已存在", err, ctx)
				return
			}
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
		_, err = db.Collection(UserModelName).InsertOne(ctx, &userModel)
		if err != nil {
			IrisRespErr("新增用户失败", err, ctx, 500)
			return
		}

		// 生成token
		jwtResp := pipe.JwtGen.Run(ctx, &pipe.PipeJwtDep{
			Env:    pipe.CtxGetEnv(ctx),
			UserId: userModel.Uid,
		}, &pipe.JwtGenPipe{}, rdb)

		if jwtResp.Err != nil {
			IrisRespErr("生成登录令牌失败", jwtResp.Err, ctx, jwtResp.ReqCode)
			return
		}

		ctx.JSON(iris.Map{"token": jwtResp.Result})
	}
}

// jwt相关?
