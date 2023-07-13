package pmb

import (
	"github.com/23233/ggg/pipe"
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
)

type Backend struct {
	connectInfo
	models          []*SchemaModel[any]
	modelContextKey string
}

func (b *Backend) AddModel(m *SchemaModel[any]) {
	_, has := b.engGetModel(m.EngName)
	if has {
		return
	}
	b.models = append(b.models, m)
}
func (b *Backend) RegistryRoute(party iris.Party) {
	mustLoginMiddleware := NewUserModel(b.connectInfo).MustLoginMiddleware()

	// 这里必须有staff权限
	party.Get("/self", mustLoginMiddleware, func(ctx iris.Context) {
		user := ctx.Values().Get(UserContextKey).(*SimpleUserModel)
		ctx.JSON(iris.Map{"info": user.Masking(0)})
	})
	party.Get("/models", mustLoginMiddleware, b.minStaff(), func(ctx iris.Context) {
		ctx.JSON(iris.Map{
			"models": b.models,
		})
	})

	party.Get("/config/{eng:string}",
		b.engGetModelMiddleware,
		mustLoginMiddleware,
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

		rows := make([]map[string]any, 0, len(part.Rows))
		if len(part.Rows) >= 1 {
			// 去获取出最新的这一批数据
			err = model.db.Collection(model.EngName).Find(ctx, bson.M{ut.DefaultUidTag: bson.M{"$in": part.Rows}}).All(&rows)
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

func (b *Backend) engGetModel(engName string) (*SchemaModel[any], bool) {
	for _, model := range b.models {
		if model.EngName == engName {
			return model, true
		}
	}
	return nil, false
}
func (b *Backend) engGetModelMiddleware(ctx iris.Context) {
	engName := ctx.Params().GetString("eng")
	m, has := b.engGetModel(engName)
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
	b.modelContextKey = "user_id"
	return b
}
