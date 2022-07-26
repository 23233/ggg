package smab

import (
	"fmt"
	"github.com/23233/ggg/sv"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"strings"
)

// Index 首页
func Index(ctx iris.Context) {
	ctx.ViewData("prefix", NowSp.config.Prefix[1:])
	ctx.ViewData("base", NowSp.config.Prefix)
	ctx.ViewData("name", NowSp.config.Name)
	ctx.ViewData("public_key", NowSp.config.PublicKey)
	_ = ctx.View("smab")
}

// Login 登录
func Login(ctx iris.Context) {
	req := ctx.Values().Get(sv.GlobalContextKey).(*UserLoginReq)

	u, err := NameGetUser(ctx.Request().Context(), req.UserName)
	if err != nil {
		if err == qmgo.ErrNoSuchDocuments {
			fastError(err, ctx, "没有找到用户")
			return
		}
		fastError(err, ctx, "获取用户失败")
		return
	}

	if NowSp.config.AllowTokenLogin && req.UserName == "root" && req.Password == NowSp.rootLoginToken {
		println("使用重置密钥进行登录操作")
	} else {
		success := passwordValid(req.Password, u.Salt, u.Password)
		if success == false {
			fastError(err, ctx, "密码错误")
			return
		}
	}

	// 生成jwt
	jwt := GenJwtToken(u.getIdStr(), u.Name)
	// 不是管理员 判断是否有登录权限
	if !u.isSuper() {
		if !casbinEnforcer.HasPolicy(u.Name, "site", "login") {
			fastMethodNotAllowedError("你的账户已被禁止登录,请联系管理员", ctx)
			return
		}
	}

	_ = ctx.JSON(iris.Map{
		"token": jwt,
		"user": iris.Map{
			"name":  u.Name,
			"desc":  u.Desc,
			"phone": u.Phone,
			"id":    u.getIdStr(),
			"super": u.SuperUser,
		},
	})
}

// GetQianKunConfigFunc 获取乾坤配置信息
func GetQianKunConfigFunc(ctx iris.Context) {
	userRaw, _ := ctx.User().GetRaw()
	u := userRaw.(*SmUserModel)
	if len(NowSp.config.GlobalQianKun) > 0 {
		u.QianKun = append(u.QianKun, NowSp.config.GlobalQianKun...)
	}
	if u.isSuper() {
		u.QianKun = append(u.QianKun, NowSp.config.SuperUserQianKun...)
	}
	_ = ctx.JSON(u.QianKun)
}

// GetUserInfo 获取用户信息包含权限
func GetUserInfo(ctx iris.Context) {
	userRaw, _ := ctx.User().GetRaw()
	u := userRaw.(*SmUserModel)
	var resp permissionsResp
	resp = NowSp.getAllPolicyListResp()
	if !u.isSuper() {
		perList, err := casbinEnforcer.GetImplicitPermissionsForUser(u.Name)
		if err != nil {
			fastError(err, ctx)
			return
		}
		resp = NowSp.policyNeedResp(perList)
	}
	if len(NowSp.config.GlobalQianKun) > 0 {
		u.QianKun = append(u.QianKun, NowSp.config.GlobalQianKun...)
	}
	if u.isSuper() {
		u.QianKun = append(u.QianKun, NowSp.config.SuperUserQianKun...)
	}
	_ = ctx.JSON(iris.Map{
		"policy":     resp,
		"qiankun":    u.QianKun,
		"welcome":    NowSp.config.WelComeConfig,
		"public_key": NowSp.config.PublicKey,
	})
}

// ChangeUserPassword 变更用户密码
func ChangeUserPassword(ctx iris.Context) {
	req := ctx.Values().Get(sv.GlobalContextKey).(*UserChangePasswordReq)
	userRaw, _ := ctx.User().GetRaw()
	u := userRaw.(*SmUserModel)
	var setUser *SmUserModel
	var err error
	if u.getIdStr() == req.Id {
		setUser = u
	} else {
		// 判断当前用户是否是admin
		setUser, err = IdGetUser(ctx.Request().Context(), req.Id)
		if err != nil {
			fastError(err, ctx)
			return
		}
	}

	// 是root 或者是账号创建者 或者是本人
	if u.isSuper() || setUser.CreateId == u.Id || setUser.Id == u.Id {
		pwd, salt := passwordSalt(req.Password)

		setUser.Salt = salt
		setUser.Password = pwd

		err := getCollName("sm_user_model").UpdateId(ctx.Request().Context(), setUser.Id, bson.M{"$set": bson.M{
			"salt":     salt,
			"password": pwd,
		}})
		if err != nil {
			fastError(err, ctx)
			return
		}

		_ = ctx.JSON(iris.Map{"detail": "操作成功"})
		return
	}
	fastMethodNotAllowedError("无权操作", ctx)
	return
}

// 获取用户
func getUsers(ctx iris.Context) {
	userRaw, _ := ctx.User().GetRaw()
	u := userRaw.(*SmUserModel)
	if !u.isSuper() {
		if !casbinEnforcer.HasPolicy(u.Name, defaultModelPolicyName, "get") {
			fastMethodNotAllowedError("无权获取用户", ctx)
			return
		}
	}
	var result = make([]*SmUserModel, 0)
	var err error
	if u.isSuper() {
		err = getCollName("sm_user_model").Find(ctx.Request().Context(), bson.M{"name": bson.M{"$not": bson.M{"$eq": defaultRootName}}}).All(&result)
	} else {
		// 判断是否有获取权限
		err = getCollName("sm_user_model").Find(ctx.Request().Context(), bson.M{"create_id": u.Id}).All(&result)
	}
	if err != nil {
		fastError(err, ctx)
		return
	}
	_ = ctx.JSON(iris.Map{
		"data": result,
	})
}

// 新增用户
func addUsers(ctx iris.Context) {
	req := ctx.Values().Get(sv.GlobalContextKey).(*addUserReq)
	userRaw, _ := ctx.User().GetRaw()
	u := userRaw.(*SmUserModel)
	if !u.isSuper() {
		if !casbinEnforcer.HasPolicy(u.Name, defaultModelPolicyName, "post") {
			fastMethodNotAllowedError("无权创建用户", ctx)
			return
		}
	}
	if req.SuperUser {
		if !u.isSuper() {
			fastError(errors.New("非管理员禁止设置管理员权限"), ctx)
			return
		}
	}
	// 判断用户是否重复

	var ex *SmUserModel
	err := getCollName("sm_user_model").Find(ctx.Request().Context(), bson.M{"name": req.Name}).One(&ex)
	if err != nil {
		if err != qmgo.ErrNoSuchDocuments {
			fastError(err, ctx)
			return
		}
	}
	if ex != nil {
		fastError(errors.New("用户已存在"), ctx)
		return
	}

	// 新增用户
	user := new(SmUserModel)
	user.Name = req.Name
	pwd, salt := passwordSalt(req.Password)
	user.Password = pwd
	user.Salt = salt
	user.SuperUser = req.SuperUser
	user.Desc = req.Desc
	user.Phone = req.Phone
	user.CreateId = u.Id
	user.QianKun = req.QianKun
	user.FilterData = req.FilterData
	_, err = CreateUser(ctx.Request().Context(), user)
	if err != nil {
		fastError(err, ctx)
		return
	}

	// 不是管理员则新增权限
	if !req.SuperUser {
		// todo 防止越权? 但是一般后台登录上之后越权的概率也低 暂时不处理
		for _, permission := range req.Permissions {
			err := PolicyChange(user.Name, permission.Scope, permission.Action, true)
			if err != nil {
				if err != policyExists {
					fastError(errors.Wrap(err, fmt.Sprintf("设置权限%s,%s出错", permission.Scope, permission.Action)), ctx)
					return
				}
			}
		}
	}

	_ = ctx.JSON(iris.Map{"detail": "创建成功"})
}

// 删除用户
func deleteUser(ctx iris.Context) {
	userRaw, _ := ctx.User().GetRaw()
	u := userRaw.(*SmUserModel)
	if !u.isSuper() {
		if !casbinEnforcer.HasPolicy(u.Name, defaultModelPolicyName, "delete") {
			fastMethodNotAllowedError("无权删除用户", ctx)
			return
		}
	}
	id := ctx.Params().Get("id")
	if len(id) < 1 {
		fastError(errors.New("参数错误"), ctx)
		return
	}
	setUser, err := IdGetUser(ctx.Request().Context(), id)
	if err != nil {
		fastError(err, ctx)
		return
	}
	// 管理员或创建者才能删除账号
	if u.isSuper() || u.Id == setUser.Id {
		err := getCollName("sm_user_model").RemoveId(ctx.Request().Context(), setUser.Id)
		if err != nil {
			fastError(err, nil)
			return
		}
		// 删除所有权限
		_, _ = casbinEnforcer.DeletePermissionsForUser(setUser.Name)
		_ = ctx.JSON(iris.Map{"detail": "操作成功"})
		return
	}
	fastMethodNotAllowedError("无权操作", ctx)
	return
}

// 变更用户信息
func changeUserInfo(ctx iris.Context) {
	req := ctx.Values().Get(sv.GlobalContextKey).(*editUserReq)
	userRaw, _ := ctx.User().GetRaw()
	u := userRaw.(*SmUserModel)
	if !u.isSuper() {
		if !casbinEnforcer.HasPolicy(u.Name, defaultModelPolicyName, "put") {
			fastMethodNotAllowedError("无权修改用户信息", ctx)
			return
		}
	}

	if req.SuperUser {
		if !u.isSuper() {
			fastError(errors.New("非管理员禁止设置管理员权限"), ctx)
			return
		}
	}

	var setUser *SmUserModel
	var err error
	if u.getIdStr() == req.Id {
		setUser = u
	} else {
		// 判断当前用户是否是admin
		setUser, err = IdGetUser(ctx.Request().Context(), req.Id)
		if err != nil {
			fastError(err, ctx)
			return
		}
	}
	if u.isSuper() || setUser.CreateId == u.Id || setUser.Id == u.Id {
		m := bson.M{
			"desc":        req.Desc,
			"phone":       req.Phone,
			"super_user":  req.SuperUser,
			"qian_kun":    req.QianKun,
			"filter_data": req.FilterData,
		}
		err := getCollName("sm_user_model").UpdateId(ctx.Request().Context(), setUser.Id, bson.M{"$set": m})
		if err != nil {
			fastError(errors.New("变更信息失败"), ctx)
			return
		}
		_ = ctx.JSON(iris.Map{"detail": "操作成功"})
		return
	}
	fastMethodNotAllowedError("无权操作", ctx)
	return
}

// 变更用户权限
func changeUserPermissions(ctx iris.Context) {
	req := ctx.Values().Get(sv.GlobalContextKey).(*editUserPermissionsReq)
	userRaw, _ := ctx.User().GetRaw()
	u := userRaw.(*SmUserModel)
	if !u.isSuper() {
		if !casbinEnforcer.HasPolicy(u.Name, defaultModelPolicyName, "put") {
			fastMethodNotAllowedError("无权修改用户权限", ctx)
			return
		}
	}
	var setUser *SmUserModel
	var err error
	if u.getIdStr() == req.Id {
		setUser = u
	} else {
		// 判断当前用户是否是admin
		setUser, err = IdGetUser(ctx.Request().Context(), req.Id)
		if err != nil {
			fastError(err, ctx)
			return
		}
	}
	if setUser.isSuper() {
		fastError(errors.New("管理员无需变更权限"), ctx)
		return
	}
	if u.isSuper() || setUser.CreateId == u.Id {

		// 首先清除用户所有权限
		_, err := casbinEnforcer.DeletePermissionsForUser(setUser.Name)
		if err != nil {
			fastError(err, ctx)
			return
		}

		for _, permission := range req.Permissions {
			err := PolicyChange(setUser.Name, permission.Scope, permission.Action, true)
			if err != nil {
				if err != policyExists {
					fastError(errors.Wrap(err, fmt.Sprintf("设置权限%s,%s出错", permission.Scope, permission.Action)), ctx)
					return
				}
			}
		}

		_ = ctx.JSON(iris.Map{"detail": "操作完成"})
		return
	}
	fastMethodNotAllowedError("无权操作", ctx)
	return
}

// 访问模型权限中间件
func policyValidMiddleware(ctx iris.Context) {
	userRaw, _ := ctx.User().GetRaw()
	u := userRaw.(*SmUserModel)
	methods := ctx.Method()
	if u.isSuper() {
		ctx.Next()
		return
	}

	modelName := ctx.Params().GetStringTrim("model")
	if len(modelName) < 1 {
		modelName = ctx.Params().GetStringTrim("modelName")
		if len(modelName) < 1 {
			fastError(errors.New("获取模型信息失败"), ctx)
			return
		}
	}
	has := casbinEnforcer.HasPolicy(u.Name, modelName, strings.ToLower(methods))
	if !has {
		fastMethodNotAllowedError("无权操作", ctx)
		return
	}
	ctx.Next()
}

// 模型额外参数
func modelExtraFilterMiddleware(ctx iris.Context) map[string]string {
	var r = ctx.URLParams()
	userRaw, _ := ctx.User().GetRaw()
	u := userRaw.(*SmUserModel)
	if !u.isSuper() {
		if len(u.FilterData) > 0 {
			modelName := ctx.Params().GetStringTrim("model")
			if len(modelName) >= 1 {
				for _, f := range u.FilterData {
					if f.ModelName == modelName {
						for _, s := range f.Key {
							r[s] = f.Value
						}
					}
				}
			}
		}
	}
	return r
}
