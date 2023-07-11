package pipe

import (
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
)

// RbacGetRolePipe 获取用户角色操作序列包
type RbacGetRolePipe struct {
	Domain string `json:"domain,omitempty"`
	UserId string `json:"user_id,omitempty"`
}

// RbacAllowPipe 用户是否有权通过此操作
// 谁(sub)在那个域名(domain)下进行了什么资源(obj)的什么操作(act)
type RbacAllowPipe struct {
	Sub    string `json:"sub,omitempty"`    // 访问对象不能为空
	Obj    string `json:"obj,omitempty"`    // 默认当前访问路径
	Domain string `json:"domain,omitempty"` // 默认自身站点名称
	Act    string `json:"act,omitempty"`    // 默认当前请求方式
}

var (
	RbacRoles = &RunnerContext[any, *RbacGetRolePipe, *RbacDomain, []string]{
		Key:  "rbac_get_role",
		Name: "rbac权限获取",
		call: func(ctx iris.Context, origin any, params *RbacGetRolePipe, db *RbacDomain, more ...any) *RunResp[[]string] {
			roles, err := db.E.GetRolesForUser(params.UserId, params.Domain)
			if err != nil {
				return newPipeErr[[]string](err)
			}
			return newPipeResult(roles)
		},
	}
	RbacAllow = &RunnerContext[any, *RbacAllowPipe, *RbacDomain, bool]{
		Key:  "rbac_allow",
		Name: "rbac权限允许执行",
		call: func(ctx iris.Context, origin any, params *RbacAllowPipe, db *RbacDomain, more ...any) *RunResp[bool] {
			if len(params.Sub) < 1 {
				return newPipeErr[bool](errors.New("权限判断操作者不能为空"))
			}
			// 访问资源为空的话 则是默认当前请求访问路径
			if len(params.Obj) < 1 {
				params.Obj = ctx.Path()
			}
			// 未传入domain 则是自身站点信息
			if len(params.Domain) < 1 {
				params.Domain = RbacSelfDomainName
			}
			// 未传入act则默认为当前请求方式
			if len(params.Act) < 1 {
				params.Act = ctx.Method()
			}
			// 谁(sub)在那个域名(domain)下进行了什么资源(obj)的什么操作(act)
			pass := db.E.HasPolicy(params.Sub, params.Domain, params.Obj, params.Act)
			return newPipeResult(pass)
		},
	}
)
