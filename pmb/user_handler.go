package pmb

import (
	"context"
	"github.com/23233/ggg/pipe"
	"github.com/23233/ggg/sv"
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
)

type UserPasswordLoginReq struct {
	UserName   string `json:"user_name,omitempty" comment:"用户名" validate:"required,min=3,max=24"`
	Password   string `json:"password,omitempty" comment:"密码" validate:"required,min=6,max=36"`
	ValidId    string `json:"valid_id,omitempty" comment:"验证码id"`
	ValidValue string `json:"valid_value,omitempty" comment:"验证码值"`
	Force      bool   `json:"force,omitempty" comment:"强制"`
	Strict     bool   `json:"strict,omitempty" comment:"严苛模式"`
}

type UserChangePasswordReq struct {
	OldPassword string `json:"old_password,omitempty" comment:"旧密码" validate:"required,min=6,max=36"`
	Password    string `json:"password,omitempty" comment:"新密码" validate:"required,min=6,max=36"`
	ValidId     string `json:"valid_id,omitempty" comment:"验证码id"`
	ValidValue  string `json:"valid_value,omitempty" comment:"验证码值"`
}

type UserRegLoginReq struct {
	UserName   string `json:"user_name,omitempty" comment:"用户名" validate:"required,min=3,max=24"`
	Password   string `json:"password,omitempty" comment:"密码" validate:"required,min=6,max=36"`
	Invite     string `json:"invite,omitempty" comment:"邀请人" validate:"omitempty,max=50"`
	ValidId    string `json:"valid_id,omitempty" comment:"验证码id"`
	ValidValue string `json:"valid_value,omitempty" comment:"验证码值"`
}

type EmailPasswordLoginReq struct {
	Email      string `json:"email,omitempty" comment:"邮箱号码" validate:"required,email"`
	Password   string `json:"password,omitempty" comment:"密码" validate:"required,min=6,max=36"`
	Force      bool   `json:"force,omitempty" comment:"强制"`
	Strict     bool   `json:"strict,omitempty" comment:"严苛模式"`
	ValidId    string `json:"valid_id,omitempty" comment:"验证码id"`
	ValidValue string `json:"valid_value,omitempty" comment:"验证码值"`
}

type RoleUpLoginReq struct {
	Id     string `json:"id" comment:"id" validate:"required,max=50"`
	Secret string `json:"secret" comment:"秘钥" validate:"required,max=50"`
	Role   string `json:"role" comment:"权限" validate:"required,max=50"`
}

func (c *SimpleUserModel) GetCollName() string {
	return UserModelName
}

func (c *SimpleUserModel) SyncIndex(ctx context.Context) error {
	cl, err := c.db.Collection(UserModelName).CloneCollection()
	if err != nil {
		return err
	}
	err = ut.MCreateIndex(ctx, cl,
		ut.MGenUnique(ut.DefaultUidTag, false),
		ut.MGenUnique("user_name", true),
		ut.MGenNormal("tel_phone"),
		ut.MGenUnique("email", true),
		ut.MGenNormal("platforms.appid"),
		ut.MGenNormal("platforms.name"),
		ut.MGenNormal("platforms.pid"),
		ut.MGenNormal("platforms.union_id"),
	)
	return err
}
func (c *SimpleUserModel) LoginUseUserNameHandler(useValid bool) iris.Handler {
	return func(ctx iris.Context) {
		var body = new(UserPasswordLoginReq)
		err := ctx.ReadBody(&body)
		if err != nil {
			IrisRespErr("解析请求包参数错误", err, ctx)
			return
		}
		err = sv.GlobalValidator.Check(body)
		if err != nil {
			IrisRespErr("", err, ctx)
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

		// 判断验证码
		if useValid {
			// 判断验证码
			if body.ValidId == "" || body.ValidValue == "" {
				IrisRespErr("未获取到验证码", nil, ctx)
				return
			}
			// 进行验证
			_, err = ImgCaptchaInst.Verify(body.ValidId, body.ValidValue)
			if err != nil {
				IrisRespErr("", err, ctx)
				return
			}
		}

		userModel, err := c.GetUserItem(ctx, bson.M{"user_name": body.UserName})
		if err != nil {
			IrisRespErr("用户名或密码有误", err, ctx)
			return
		}

		c.passwordLogin(ctx, "用户名", userModel, body.Password, body.Force, body.Strict)
	}
}

func (c *SimpleUserModel) ChangePassword(useValid bool) iris.Handler {
	return func(ctx iris.Context) {
		var body = new(UserChangePasswordReq)
		err := ctx.ReadBody(&body)
		if err != nil {
			IrisRespErr("解析请求包参数错误", err, ctx)
			return
		}
		err = sv.GlobalValidator.Check(body)
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}
		// 判断验证码
		if useValid {
			// 判断验证码
			if body.ValidId == "" || body.ValidValue == "" {
				IrisRespErr("未获取到验证码", nil, ctx)
				return
			}
			// 进行验证
			_, err = ImgCaptchaInst.Verify(body.ValidId, body.ValidValue)
			if err != nil {
				IrisRespErr("", err, ctx)
				return
			}
		}
		user := ctx.Values().Get(UserContextKey).(*SimpleUserModel)
		// 判断密码是否正确
		if !validPassword(body.OldPassword, user.Salt, user.Password) {
			IrisRespErr("密码错误", nil, ctx)
			return
		}
		// 密码正确就进行密码的修改
		saltPassword, salt := passwordSalt(body.Password)
		err = c.db.Collection(UserModelName).UpdateId(ctx, user.Id, bson.M{
			"$set": bson.M{
				"password": saltPassword,
				"salt":     salt,
			},
		})
		if err != nil {
			IrisRespErr("修改密码失败", err, ctx)
			return
		}
		// 修改成功了要不要下线呢?
		ctx.JSON(iris.Map{"detail": "修改成功"})
	}
}

func (c *SimpleUserModel) LoginUseEmailHandler(useValid bool) iris.Handler {
	return func(ctx iris.Context) {
		var body = new(EmailPasswordLoginReq)
		err := ctx.ReadBody(&body)
		if err != nil {
			IrisRespErr("解析请求包参数错误", err, ctx)
			return
		}
		err = sv.GlobalValidator.Check(body)
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}
		rateResp := pipe.RequestRate.Run(ctx, nil, &pipe.RateLimitPipe{
			RatePeriod: "5-M",
			KeyGen: &pipe.StrExpand{
				Key: "lf:${un}",
				KeyMap: []pipe.StrTemplate{
					{
						VarName: "un",
						Value:   body.Email,
					},
				},
			},
			WriteHeader: false,
		}, nil)
		if rateResp.Err != nil {
			IrisRespErr("操作过快", rateResp.Err, ctx, rateResp.ReqCode)
			return
		}
		// 判断验证码
		if useValid {
			// 判断验证码
			if body.ValidId == "" || body.ValidValue == "" {
				IrisRespErr("未获取到验证码", nil, ctx)
				return
			}
			// 进行验证
			_, err = ImgCaptchaInst.Verify(body.ValidId, body.ValidValue)
			if err != nil {
				IrisRespErr("", err, ctx)
				return
			}
		}

		userModel, err := c.GetUserItem(ctx, bson.M{"email": body.Email})
		if err != nil {
			IrisRespErr("邮箱错误", err, ctx)
			return
		}

		c.passwordLogin(ctx, "邮箱", userModel, body.Password, body.Force, body.Strict)

	}
}
func (c *SimpleUserModel) RoleSetHandler() iris.Handler {
	return func(ctx iris.Context) {
		var body = new(RoleUpLoginReq)
		err := ctx.ReadBody(&body)
		if err != nil {
			IrisRespErr("解析请求包参数错误", err, ctx)
			return
		}
		err = sv.GlobalValidator.Check(body)
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}
		// 判断秘钥是否一致
		if body.Secret != c.RoleSecret() {
			IrisRespErr("秘钥错误", err, ctx)
			return
		}
		err = c.SetRole(ctx, bson.M{
			"$or": bson.A{
				bson.D{{"user_name", body.Id}},
				bson.D{{"uid", body.Id}},
			},
		}, body.Role)
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}
		ctx.JSON(iris.Map{})

		MustOpLog(ctx, c.OpLog(), "role", nil, "role", "设置权限", body.Id, []ut.Kov{{Key: "role", Value: body.Role}})

		return
	}
}
func (c *SimpleUserModel) passwordLogin(ctx iris.Context, event string, user *SimpleUserModel, password string, force bool, strict bool) {

	// 验证密码是否正确
	if !validPassword(password, user.Salt, user.Password) {
		IrisRespErr(event+"或密码错误", nil, ctx)
		return
	}

	// 正确的情况下 判断是否可以登录
	disableLoginResp := pipe.RbacAllow.Run(ctx, nil, &pipe.RbacAllowPipe{
		Sub:    user.Uid,
		Obj:    pipe.RbacNotAllowLoginObj,
		Domain: pipe.RbacSelfDomainName,
		Act:    pipe.RbacNormalAct,
	}, c.rbac)

	// 在这个组里就是不允许登录的
	if disableLoginResp.Result {
		IrisRespErr("该用户被禁止登录", nil, ctx)
		return
	}

	token, err := user.GenJwtToken(ctx, force, strict)
	if err != nil {
		IrisRespErr("生成登录令牌失败", err, ctx)
		return
	}

	ctx.JSON(iris.Map{"token": token, "info": user.Masking(0)})

	// 写入日志
	MustOpLog(ctx, c.OpLog(), "login", user, "user", event+"登录成功", "", nil)

}
func (c *SimpleUserModel) RegistryUseUserNameHandler(useValid bool) iris.Handler {
	return func(ctx iris.Context) {
		var body = new(UserRegLoginReq)
		err := ctx.ReadBody(&body)
		if err != nil {
			IrisRespErr("解析请求包参数错误", err, ctx)
			return
		}
		// 判断验证码
		if useValid {
			// 判断验证码
			if body.ValidId == "" || body.ValidValue == "" {
				IrisRespErr("未获取到验证码", nil, ctx)
				return
			}
			// 进行验证
			_, err = ImgCaptchaInst.Verify(body.ValidId, body.ValidValue)
			if err != nil {
				IrisRespErr("", err, ctx)
				return
			}
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

		// 如果邀请人存在
		if body.Invite != "" {
			// 判断Invite是否为objectId
			fl := bson.M{}
			oid, _ := primitive.ObjectIDFromHex(body.Invite)
			if !oid.IsZero() {
				fl["_id"] = oid
			} else {
				fl[ut.DefaultUidTag] = body.Invite
			}
			// 获取这个邀请人
			yqrUser, err := c.GetUserItem(ctx, fl)
			if err != nil {
				IrisRespErr("邀请人不存在", err, ctx)
				return
			}
			userModel.ReferrerUid = yqrUser.Uid

		}

		// 进行注册
		password, salt := passwordSalt(body.Password)
		userModel.UserName = body.UserName
		userModel.Password = password
		userModel.ReferrerUid = body.Invite
		userModel.Salt = salt
		userModel.NickName = ut.RandomStr(12)
		_ = userModel.BeforeInsert(ctx)

		// 插入用户
		_, err = c.db.Collection(UserModelName).InsertOne(ctx, &userModel)
		if err != nil {
			IrisRespErr("新增用户失败", err, ctx, 500)
			return
		}
		userModel.connectInfo = c.connectInfo
		token, err := userModel.GenJwtToken(ctx, false, false)
		if err != nil {
			IrisRespErr("生成登录令牌失败", err, ctx)
			return
		}
		ctx.JSON(iris.Map{"token": token, "info": userModel.Masking(0)})

		MustOpLog(ctx, c.OpLog(), "reg", userModel, "user", "用户名密码注册", "", nil)

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
		ctx.Values().Set(UserIdContextKey, userModel.Uid)
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
func (c *SimpleUserModel) FuzzGetUser(ctx context.Context, idOrName string) (*SimpleUserModel, error) {
	return c.GetUserItem(ctx, bson.M{
		"$or": bson.A{
			bson.D{{"user_name", idOrName}},
			bson.D{{"uid", idOrName}},
		},
	})
}

func (c *SimpleUserModel) GenJwtToken(ctx iris.Context, force bool, strict bool) (string, error) {
	// 生成token
	jwtResp := pipe.JwtGen.Run(ctx, &pipe.PipeJwtDep{
		Env:    pipe.CtxGetEnv(ctx),
		UserId: c.Uid,
	}, &pipe.JwtGenPipe{
		Force:  force,
		Strict: strict,
	}, c.rdb)
	return jwtResp.Result, jwtResp.Err
}

func (c *SimpleUserModel) SetRoleUseUserName(ctx context.Context, userName string, roleTarget string) error {
	return c.SetRole(ctx, bson.M{"user_name": userName}, roleTarget)
}
func (c *SimpleUserModel) SetRoleUseEmail(ctx context.Context, email string, roleTarget string) error {
	return c.SetRole(ctx, bson.M{"email": email}, roleTarget)
}
func (c *SimpleUserModel) SetRoleUsePhone(ctx context.Context, phone string, roleTarget string) error {
	return c.SetRole(ctx, bson.M{"tel_phone": phone}, roleTarget)
}
func (c *SimpleUserModel) SetRoleUseUid(ctx context.Context, uid string, roleTarget string) error {
	return c.SetRole(ctx, bson.M{"uid": uid}, roleTarget)
}
func (c *SimpleUserModel) SetRole(ctx context.Context, filters bson.M, roleTarget string) error {
	user, err := c.GetUserItem(ctx, filters)
	if err != nil {
		return err
	}

	_, err = c.rbac.SetRoleDefaultDomain(user.Uid, roleTarget)

	return err
}
func (c *SimpleUserModel) CanRole(ctx context.Context, uid string, roles []string) bool {
	// 判断roles中是否有root 如果没有则加上 root应该是必须的
	if !ArrayIn("root", roles) {
		roles = append(roles, "root")
	}

	return c.rbac.HasRoles(uid, roles)
}

func (c *SimpleUserModel) RemoveUser(ctx context.Context, uid string) error {
	return c.db.Collection(UserModelName).Remove(ctx, bson.M{ut.DefaultUidTag: uid})
}
