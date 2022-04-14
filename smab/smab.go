package smab

import (
	"embed"
	"github.com/23233/ggg/mab"
	"github.com/23233/ggg/sv"
	"github.com/imdario/mergo"
	"github.com/kataras/iris/v12"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
)

var NowSp *SpAdmin

var defaultModelPolicyName = "mab_user"

//go:embed dist/*
var embedWeb embed.FS

type SpAdmin struct {
	Mdb            *qmgo.Database
	mab            *mab.RestApi
	config         *Configs
	rootLoginToken string
}

func New(mdb *qmgo.Database, c Configs) (*SpAdmin, error) {
	// 合并配置文件
	newConf := c.initConfig()
	if err := mergo.Map(&c, newConf); err != nil {
		return nil, err
	}
	// 验证配置是否正确
	if err := c.valid(); err != nil {
		return nil, err
	}

	nowSpAdmin := &SpAdmin{
		config: &c,
		Mdb:    mdb,
	}
	NowSp = nowSpAdmin

	// 注册视图
	NowSp.register()

	// 初始化权限
	NowSp.initCasBin()

	// 初始化管理员
	NowSp.initSuperUser()

	return NowSp, nil
}

// 在这里注册主路由
func (lib *SpAdmin) router(router iris.Party) {
	fsys := iris.PrefixDir("dist", http.FS(embedWeb))
	router.RegisterView(iris.Blocks(fsys, ".html"))

	// 首页
	router.Get("/", Index)
	// 登录
	router.Post("/login", sv.Run(new(UserLoginReq)), Login)

	v := router.Party("/v", CustomJwt.Serve, TokenToUserUidMiddleware, idGetUserMiddleware)
	// 获取用户所有权限
	v.Get("/user_info", GetUserInfo)
	v.Get("/qiankun", GetQianKunConfigFunc)

	if lib.config.OnFileUpload != nil {
		v.Post("/file_upload", lib.config.OnFileUpload)
	}

	// 变更用户密码
	v.Post("/change_password", sv.Run(new(UserChangePasswordReq)), ChangeUserPassword)
	// 获取用户 管理员全部 否则为自己创建的用户
	v.Get("/self_users", getUsers)
	// 创建用户
	v.Post("/self_users", sv.Run(new(addUserReq)), addUsers)
	// 删除用户
	v.Delete("/self_users/{id:string}", deleteUser)
	// 变更用户信息
	v.Put("/self_users", sv.Run(new(editUserReq)), changeUserInfo)
	v.Put("/self_users_permissions", sv.Run(new(editUserPermissionsReq)), changeUserPermissions)

	mab.New(&mab.Config{
		Party: v,
		Mdb:   lib.Mdb,
		Models: []*mab.SingleModel{
			{
				Model:        new(SmTask),
				AllowMethods: []string{"get(all)", "put", "delete"},
				GetAllMustFilters: map[string]string{
					"to_user": "",
				},
				AllowGetInfo: true,
				ShowCount:    true,
				ShowDocCount: true,
				MustSearch:   true,
			},
			{
				Model: new(SmDashBoardScreen),
				Pk: func() []bson.D {
					result := make([]bson.D, 0)
					dashBoardLook := bson.D{{"$lookup", bson.D{
						{"from", "sm_dash_board"},
						{"localField", "_id"},
						{"foreignField", "screen_id"},
						{"as", "dash_board"},
					}}}
					userLook := bson.D{{"$lookup", bson.D{
						{"from", "sm_user_model"},
						{"localField", "view_user_id"},
						{"foreignField", "_id"},
						{"as", "view_users"},
					}}}
					result = append(result, dashBoardLook, userLook)
					return result
				},
				GetAllMustFilters: map[string]string{
					"create_user_id": "",
				},
				PostMustFilters: map[string]string{
					"create_user_id": "",
				},
				AllowGetInfo: true,
			},
			{
				Model:        new(SmDashBoard),
				AllowGetInfo: true,
			},
			{
				Model: new(SmAction),
				GetAllMustFilters: map[string]string{
					"user_id": "",
				},
				PostMustFilters: map[string]string{
					"create_user_id": "",
				},
				AllowGetInfo: true,
			},
		},
	})

	// 权限验证middleware
	c := v.Party("/c", policyValidMiddleware)

	var models = make([]*mab.SingleModel, 0, len(lib.config.ModelList))
	for _, m := range lib.config.ModelList {
		models = append(models, &mab.SingleModel{
			Model:        m,
			ShowDocCount: true,
			ShowCount:    true,
			AllowGetInfo: true,
			MustSearch:   true,
			InjectParams: modelExtraFilterMiddleware,
		})
	}
	lib.mab = mab.New(&mab.Config{
		Generator: true,
		Party:     c,
		Mdb:       lib.Mdb,
		Models:    models,
	})

	router.Get("/{root:path}", Index)

}

// 注册视图
func (lib *SpAdmin) register() {

	app := lib.config.App
	iris.WithoutBodyConsumptionOnUnmarshal(app)

	fsys := iris.PrefixDir("dist", http.FS(embedWeb))
	app.HandleDir("/smab_static", fsys) // /smab_static/umi.js
	app.PartyFunc(lib.config.Prefix, lib.router)
}
