package pmb

import (
	"context"
	"embed"
	"github.com/23233/gocaptcha"
	"net/http"
	"path"
	"strings"

	"github.com/23233/ggg/logger"
	"github.com/23233/ggg/pipe"
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/apps"
	"github.com/kataras/iris/v12/core/router"
	"github.com/pkg/errors"
	"github.com/qiniu/qmgo"
	"github.com/redis/rueidis"
)

var (
	BkInst *Backend
)

//go:embed template/*.html
var htmlWeb embed.FS

//go:embed template/assets/*
var assetsWeb embed.FS

type Backend struct {
	connectInfo
	models          []IModelItem
	modelContextKey string
	msg             *MessageQueue
	Prefix          string
	LoginUseValid   bool
	RegUseValid     bool
	ValidHard       gocaptcha.CaptchaDifficulty
}

func (b *Backend) GetModel(name string) (IModelItem, bool) {
	for _, model := range b.models {
		base := model.GetBase()
		if base.RawName == name || base.TableName == name || base.UniqueId == name || base.PathId == name {
			return model, true
		}
	}
	return nil, false
}

func (b *Backend) AddModel(m IModelItem) {
	// 初始化UniqueId为TableName
	uniqueId := m.GetTableName()

	// 查找唯一的UniqueId
	for {
		found := false
		for _, model := range b.models {
			if model.GetBase().PathId == uniqueId {
				found = true
				break
			}
		}
		if !found {
			break // 找到了唯一的UniqueId，跳出循环
		}
		// 如果UniqueId已经存在，尝试在其后添加0
		uniqueId += "0"
	}

	// 设置唯一的UniqueId
	m.SetPathId(uniqueId)
	b.models = append(b.models, m)
}
func (b *Backend) AddModelAny(raw any) IModelItem {
	m := NewSchemaModel(raw, b.db)
	b.AddModel(m)
	return m
}

func recordBodyMiddleware(ctx iris.Context) {
	if !ctx.IsRecordingBody() {
		ctx.RecordRequestBody(true)
	}
	ctx.Next()
}

func (b *Backend) RegistryRoute(party iris.Party) {
	apps.Get().Configure(iris.WithoutBodyConsumptionOnUnmarshal)

	frontParty := party.Party("/" + b.Prefix)
	apiParty := party.Party("/" + b.Prefix + "/apis")

	fsys := iris.PrefixDir("template", http.FS(htmlWeb))
	frontParty.RegisterView(iris.Blocks(fsys, ".html"))

	frontParty.HandleDir("/", assetsWeb, iris.DirOptions{
		Cache:    router.DirCacheOptions{},
		Compress: true,
	}) // ./prefix/assets/index-3fa15531.js

	// 注册role视图
	apiParty.RegisterView(iris.Blocks(fsys, ".html"))

	frontHandler := func(ctx iris.Context) {
		assertPrefix := party.GetRelPath() + b.Prefix
		ctx.ViewData("prefix", strings.ReplaceAll(assertPrefix, "//", "/"))
		ctx.ViewData("req_prefix", strings.ReplaceAll(apiParty.GetRelPath(), "//", "/"))
		_ = ctx.View("index")
	}

	frontParty.Get("/", frontHandler)
	frontParty.Get("/login", frontHandler)
	frontParty.Get("/reg", frontHandler)

	mustLoginMiddleware := UserInstance.MustLoginMiddleware()

	apiParty.Get("/self", mustLoginMiddleware, func(ctx iris.Context) {
		user := ctx.Values().Get(UserContextKey).(*SimpleUserModel)
		ctx.JSON(iris.Map{"info": user.Masking(0)})
	})

	apiParty.Get("/models", mustLoginMiddleware, func(ctx iris.Context) {
		var models = make([]IModelItem, 0)
		user := ctx.Values().Get(UserContextKey).(*SimpleUserModel)
		isRoot := b.rbac.HasRoles(user.Uid, []string{"root"})
		if isRoot {
			models = b.models
		} else {
			for _, model := range b.models {
				if b.canRole(user, model) {
					models = append(models, model)
				}
			}
		}

		ctx.JSON(iris.Map{
			"models": models,
		})
	})
	apiParty.Get("/message", mustLoginMiddleware, b.GetMsgHandler)

	apiParty.Get("/config/{unique:string}",
		mustLoginMiddleware,
		b.uniqueGetModelMiddleware,
		b.canRoleMiddleware,
		func(ctx iris.Context) {
			model := ctx.Values().Get(b.modelContextKey).(IModelItem)
			_ = ctx.JSON(model)
			return
		})
	apiParty.Post("/action/{unique:string}", mustLoginMiddleware, b.uniqueGetModelMiddleware, b.canRoleMiddleware, recordBodyMiddleware, func(ctx iris.Context) {
		user := ctx.Values().Get(UserContextKey).(*SimpleUserModel)
		model := ctx.Values().Get(b.modelContextKey).(IModelItem)
		ActionRun(ctx, model, user)
	})
	apiParty.Post("/dynamic/{unique:string}", mustLoginMiddleware, b.uniqueGetModelMiddleware, b.canRoleMiddleware, recordBodyMiddleware, func(ctx iris.Context) {
		user := ctx.Values().Get(UserContextKey).(*SimpleUserModel)
		model := ctx.Values().Get(b.modelContextKey).(IModelItem)
		DynamicRun(ctx, model, user)
	})

	// crud
	curd := apiParty.Party("/{unique:string}", mustLoginMiddleware, b.uniqueGetModelMiddleware, b.canRoleMiddleware)
	curd.Get("/{uid:string}", func(ctx iris.Context) {
		model := ctx.Values().Get(b.modelContextKey).(IModelItem)
		err := model.GetHandler(ctx, pipe.QueryParseConfig{}, pipe.ModelGetData{
			Single: true,
		}, "")
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}
	})
	curd.Get("/", func(ctx iris.Context) {
		//user := ctx.Values().Get(UserContextKey).(*SimpleUserModel)
		model := ctx.Values().Get(b.modelContextKey).(IModelItem)
		err := model.GetHandler(ctx, pipe.QueryParseConfig{}, pipe.ModelGetData{
			Single:        false,
			GetQueryCount: true,
		}, "")
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}
	})
	curd.Post("/", recordBodyMiddleware, func(ctx iris.Context) {

		user := ctx.Values().Get(UserContextKey).(*SimpleUserModel)
		model := ctx.Values().Get(b.modelContextKey).(IModelItem)

		// 这个模型必须包含了 UserIdFieldName 字段才注入 否则不注入
		injectData := make(map[string]any)
		if model.HaveUserKey(model.GetSchema(SchemaModeAdd)) {
			injectData[UserIdFieldName] = user.Uid
		}

		err := model.PostHandler(ctx, ut.ModelCtxMapperPack{
			InjectData: injectData,
		})
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}

	})
	curd.Put("/{uid:string}", recordBodyMiddleware, func(ctx iris.Context) {
		model := ctx.Values().Get(b.modelContextKey).(IModelItem)
		err := model.PutHandler(ctx, pipe.ModelPutConfig{
			UpdateTime: true,
			// 这里虽然没有判断就注入了用户id 但是因为drop 未找到是跳过 所以无所谓
			DropKeys: []string{UserIdFieldName},
		})
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}

	})
	curd.Delete("/{uid:string}", func(ctx iris.Context) {
		model := ctx.Values().Get(b.modelContextKey).(IModelItem)
		err := model.DelHandler(ctx, pipe.ModelDelConfig{})
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}
	})
	apiParty.Get("/captcha_img", func(ctx iris.Context) {
		imgWidth := ctx.Params().GetIntDefault("width", 120)
		imgHeight := ctx.Params().GetIntDefault("height", 44)
		textSize := ctx.Params().GetInt8Default("size", 4)
		// 生成图片
		id, bt, err := ImgCaptchaInst.GetNewImg(imgWidth, imgHeight, int(textSize), b.ValidHard)
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}
		ctx.Header("Content-Type", "image/png")
		ctx.Header("X-Captcha-Id", id) // 添加验证码ID到响应头
		_, _ = ctx.Write(bt)
	})
	apiParty.Post("/login", recordBodyMiddleware, UserInstance.LoginUseUserNameHandler(b.LoginUseValid))
	apiParty.Post("/user_change_password", recordBodyMiddleware, mustLoginMiddleware, UserInstance.ChangePassword(b.LoginUseValid))
	apiParty.Get("/set_role", func(ctx iris.Context) {
		p := path.Join(apiParty.GetRelPath(), "set_role")
		ctx.ViewData("post_address", p)
		_ = ctx.View("role")
	})
	apiParty.Post("/set_role", UserInstance.RoleSetHandler())
	apiParty.Post("/reg", UserInstance.RegistryUseUserNameHandler(b.LoginUseValid))

}

func (b *Backend) uniqueGetModelMiddleware(ctx iris.Context) {
	uniqueId := ctx.Params().GetString("unique")
	m, has := b.GetModel(uniqueId)
	if !has {
		IrisRespErr("获取模型失败", nil, ctx)
		ctx.StopExecution()
		return
	}
	ctx.Values().Set(b.modelContextKey, m)
	ctx.Next()
}

func (b *Backend) canRole(user *SimpleUserModel, model IModelItem) bool {
	if b.rbac.HasRoles(user.Uid, []string{"root"}) {
		return true
	}
	role := model.GetRoles()
	var allRole = make([]string, 0)
	if role != nil {
		allRole = append(role.RoleGroup, role.NameGroup...)
	}
	base := model.GetBase()
	allRole = append(allRole, base.UniqueId, base.TableName, base.Group)
	return b.rbac.HasRoles(user.Uid, allRole)
}

func (b *Backend) canRoleMiddleware(ctx iris.Context) {
	// 获取出当前用户和模型
	user := ctx.Values().Get(UserContextKey).(*SimpleUserModel)
	model := ctx.Values().Get(b.modelContextKey).(IModelItem)

	if !b.canRole(user, model) {
		IrisRespErr("获取权限失败", nil, ctx, http.StatusMethodNotAllowed)
		ctx.StopExecution()
		return
	}
	ctx.Next()
}

func (b *Backend) InsertLogModel() {
	m := NewSchemaModel(new(OperationLog), b.db)
	m.Alias = "操作日志"
	m.TableName = "operation_log"
	b.AddModel(m)
}
func (b *Backend) InsertUserModel() {
	m := NewSchemaModel(new(SimpleUserModel), b.db)
	m.Alias = "用户表"
	m.TableName = UserModelName
	b.AddModel(m)
}

func (b *Backend) SendMsg(ctx context.Context, uid, content string) {
	user, err := UserInstance.FuzzGetUser(ctx, uid)
	if err != nil {
		logger.J.ErrorE(err, "[%s]未找到用户 发送的消息是%s ", uid, content)
		return
	}
	b.msg.Put(user.Uid, content)
}
func (b *Backend) GetMsgHandler(ctx iris.Context) {
	user := ctx.Values().Get(UserContextKey).(*SimpleUserModel)
	msg, has := b.msg.Consume(user.GetUid(), ctx.GetHeader("User-Agent"))
	if !has {
		IrisRespErr("未获取到消息", nil, ctx)
		return
	}
	ctx.JSON(msg)
}

func NewBackend() *Backend {
	b := new(Backend)
	b.modelContextKey = "now_model"
	b.msg = NewMessageQueue()
	BkInst = b
	b.Prefix = "/manager"
	b.LoginUseValid = true
	b.RegUseValid = true
	b.ValidHard = gocaptcha.CaptchaVeryEasy
	return b
}

func NewFullBackend(party iris.Party, mongodb *qmgo.Database, redisAddress string, redisPassword string, redisDb int) (*Backend, error) {
	rdb, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{redisAddress},
		Password:    redisPassword,
		SelectDB:    redisDb,
	})
	if err != nil {
		return nil, err
	}

	err = rdb.Do(context.TODO(), rdb.B().Ping().Build()).Error()
	if err != nil {
		return nil, err
	}

	if mongodb == nil {
		return nil, errors.New("mongodb未找到连接")
	}

	bk := NewBackend()
	bk.AddDb(mongodb)
	bk.AddRdb(rdb)
	bk.AddRbacUseUri(redisAddress, redisPassword)
	bk.RegistryRoute(party)
	UserInstance.SetConn(bk.CloneConn())
	err = UserInstance.SyncIndex(context.TODO())
	if err != nil {
		logger.J.ErrorE(err, "同步用户模型索引失败")
		return nil, err
	}

	// 新增操作日志
	bk.InsertLogModel()
	bk.InsertUserModel()

	// 日志也需要索引
	go func() {
		err := OpLogSyncIndex(context.TODO(), bk.OpLog())
		if err != nil {
			logger.J.ErrorE(err, "创建操作日志索引失败")
		}
	}()

	return bk, nil

}
