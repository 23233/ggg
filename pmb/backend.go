package pmb

import (
	"embed"
	"github.com/23233/ggg/pipe"
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/router"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
	"path"
)

//go:embed template/*
var embedWeb embed.FS

type Backend struct {
	connectInfo
	models          []*SchemaModel[any]
	modelContextKey string
}

func (b *Backend) GetModel(name string) (*SchemaModel[any], bool) {
	for _, model := range b.models {
		if model.EngName == name {
			return model, true
		}
	}
	return nil, false
}
func (b *Backend) AddModel(m *SchemaModel[any]) {
	_, has := b.GetModel(m.EngName)
	if has {
		return
	}
	b.models = append(b.models, m)
}
func (b *Backend) RegistryRoute(party iris.Party) {
	fsys := iris.PrefixDir("template", http.FS(embedWeb))
	party.RegisterView(iris.Blocks(fsys, ".html"))

	party.HandleDir("/manager", fsys, iris.DirOptions{
		Cache:    router.DirCacheOptions{},
		Compress: true,
	}) // ./manager/assets/index-3fa15531.js

	party.Get("/", func(ctx iris.Context) {
		loginPath := path.Join(party.GetRelPath(), "login")

		ctx.ViewData("token_key", "ttb_token")
		ctx.ViewData("info_key", "ttb_info")
		ctx.ViewData("login_url", loginPath)
		ctx.ViewData("req_prefix", party.GetRelPath())

		prefix := party.GetRelPath()
		if prefix == "/" {
			prefix = ""
		}
		ctx.ViewData("prefix", prefix)
		_ = ctx.View("index")
	})

	mustLoginMiddleware := UserInstance.MustLoginMiddleware()

	party.Get("/self", mustLoginMiddleware, func(ctx iris.Context) {
		user := ctx.Values().Get(UserContextKey).(*SimpleUserModel)
		ctx.JSON(iris.Map{"info": user.Masking(0)})
	})
	// 这里必须有staff权限
	party.Get("/models", mustLoginMiddleware, b.minStaff(), func(ctx iris.Context) {
		ctx.JSON(iris.Map{
			"models": b.models,
		})
	})

	party.Get("/config/{eng:string}",
		mustLoginMiddleware,
		b.engGetModelMiddleware,
		b.minStaff(),
		func(ctx iris.Context) {
			model := ctx.Values().Get(b.modelContextKey).(*SchemaModel[any])
			_ = ctx.JSON(model)
			return
		})
	party.Post("/action/{eng:string}", mustLoginMiddleware, b.engGetModelMiddleware, b.minRoot(), func(ctx iris.Context) {
		user := ctx.Values().Get(UserContextKey).(*SimpleUserModel)
		model := ctx.Values().Get(b.modelContextKey).(*SchemaModel[any])

		// 必须为post
		part := new(ActionPostPart)
		err := ctx.ReadBody(&part)
		if err != nil {
			IrisRespErr("解构action参数包失败", err, ctx)
			return
		}

		action, has := model.GetAction(part.Name)
		if has == false {
			IrisRespErr("未找到对应action", nil, ctx)
			return
		}
		if action.call == nil {
			IrisRespErr("action未设置执行方法", nil, ctx)
			return
		}

		// 进行验证
		if action.Form != nil {
			resp := pipe.SchemaValid.Run(ctx, part.FormData, &pipe.SchemaValidConfig{
				Schema: action.Form,
			}, nil)
			if resp.Err != nil {
				IrisRespErr("", resp.Err, ctx)
				return
			}
		}

		rows := make([]map[string]any, 0, len(part.Rows))
		if len(part.Rows) >= 1 {
			// 去获取出最新的这一批数据
			err = model.GetCollection().Find(ctx, bson.M{ut.DefaultUidTag: bson.M{"$in": part.Rows}}).All(&rows)
			if err != nil {
				IrisRespErr("获取对应行列表失败", err, ctx)
				return
			}
		}

		result, err := action.call(ctx, rows, part.FormData, user)
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}

		if result != nil {
			_ = ctx.JSON(result)
		} else {
			_, _ = ctx.WriteString("ok")
		}

	})

	// crud
	curd := party.Party("/{eng:string}", mustLoginMiddleware, b.engGetModelMiddleware, b.minStaff())
	curd.Get("/{uid:string}", func(ctx iris.Context) {
		model := ctx.Values().Get(b.modelContextKey).(*SchemaModel[any])
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
		model := ctx.Values().Get(b.modelContextKey).(*SchemaModel[any])
		err := model.GetHandler(ctx, pipe.QueryParseConfig{}, pipe.ModelGetData{
			Single:        false,
			GetQueryCount: true,
		}, "")
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}
	})
	curd.Post("/", b.minRoot(), func(ctx iris.Context) {
		user := ctx.Values().Get(UserContextKey).(*SimpleUserModel)
		model := ctx.Values().Get(b.modelContextKey).(*SchemaModel[any])
		err := model.PostHandler(ctx, pipe.ModelCtxMapperPack{
			InjectData: map[string]any{
				"user_id": user.Uid,
			},
		})
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}
	})
	curd.Put("/{uid:string}", b.minRoot(), func(ctx iris.Context) {
		model := ctx.Values().Get(b.modelContextKey).(*SchemaModel[any])
		err := model.PutHandler(ctx, pipe.ModelPutConfig{
			// 是不是需要注入用户的id?
			UpdateTime: true,
			DropKeys:   []string{"user_id"},
		})
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}
	})
	curd.Delete("/{uid:string}", b.minRoot(), func(ctx iris.Context) {
		model := ctx.Values().Get(b.modelContextKey).(*SchemaModel[any])
		err := model.DelHandler(ctx, pipe.ModelDelConfig{})
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}
	})
}
func (b *Backend) RegistryLoginRegRoute(party iris.Party, allowReg bool) {

	party.Get("/login", func(ctx iris.Context) {
		loginPath := path.Join(party.GetRelPath(), "login")

		ctx.ViewData("post_address", loginPath)
		ctx.ViewData("allow_reg", allowReg)
		ctx.ViewData("reg_path", path.Join(party.GetRelPath(), "reg"))
		ctx.ViewData("rel_path", party.GetRelPath())
		_ = ctx.View("login")
	})
	party.Post("/login", UserInstance.LoginUseUserNameHandler())
	party.Get("/set_role", func(ctx iris.Context) {
		p := path.Join(party.GetRelPath(), "set_role")
		ctx.ViewData("post_address", p)
		_ = ctx.View("role")
	})
	party.Post("/set_role", UserInstance.RoleSetHandler())

	if allowReg {
		party.Get("/reg", func(ctx iris.Context) {
			regPath := path.Join(party.GetRelPath(), "reg")

			ctx.ViewData("login_path", path.Join(party.GetRelPath(), "login"))
			ctx.ViewData("post_address", regPath)
			ctx.ViewData("rel_path", party.GetRelPath())
			_ = ctx.View("reg")
		})
		party.Post("/reg", UserInstance.RegistryUseUserNameHandler())

	}

}
func (b *Backend) AddModelAny(raw any) *SchemaModel[any] {
	m := NewSchemaModel(raw, b.db)
	b.AddModel(m)
	return m
}

func (b *Backend) engGetModelMiddleware(ctx iris.Context) {
	engName := ctx.Params().GetString("eng")
	m, has := b.GetModel(engName)
	if !has {
		IrisRespErr("获取模型失败", nil, ctx)
		ctx.StopExecution()
		return
	}
	ctx.Values().Set(b.modelContextKey, m)
	ctx.Next()
}
func (b *Backend) gtRoleMiddleware(roles []string) iris.Handler {
	return func(ctx iris.Context) {
		user := ctx.Values().Get(UserContextKey).(*SimpleUserModel)
		if !b.rbac.HasRoles(user.Uid, roles) {
			IrisRespErr("获取权限失败", nil, ctx, http.StatusMethodNotAllowed)
			ctx.StopExecution()
			return
		}
		ctx.Next()
	}
}
func (b *Backend) minStaff() iris.Handler {
	return b.gtRoleMiddleware([]string{"staff", "root"})
}
func (b *Backend) minRoot() iris.Handler {
	return b.gtRoleMiddleware([]string{"root"})
}

func NewBackend() *Backend {
	b := new(Backend)
	b.modelContextKey = "now_model"
	return b
}