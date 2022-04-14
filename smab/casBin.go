package smab

import (
	"fmt"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	mongodbadapter "github.com/casbin/mongodb-adapter/v3"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

var casbinEnforcer *casbin.Enforcer

var defaultUserRole = []string{"get", "post", "put", "delete"}
var defaultModelRole = []string{"get", "post", "put", "delete"}
var defaultModelRoleAliasMap = map[string]string{
	"get":    "请求数据",
	"post":   "新增数据",
	"put":    "修改数据",
	"delete": "删除数据",
}
var defaultSiteRole = []string{"login"} // 站点属性 是否允许登录
var defaultSiteRoleAliasMap = map[string]string{
	"login": "登录",
}

var (
	policyExists = errors.New("权限已存在")
)

// 配置文件初始化权限
func (lib *SpAdmin) initCasBin() {
	m := model.NewModel()
	m.AddDef("r", "r", "sub, obj, act")                                                                                      // request_definition
	m.AddDef("p", "p", "sub, obj, act")                                                                                      // policy_definition
	m.AddDef("g", "g", "_, _")                                                                                               // role_definition
	m.AddDef("e", "e", "some(where (p.eft == allow))")                                                                       // policy_effect
	m.AddDef("m", "m", `g(r.sub, p.sub) && keyMatch3(r.obj, p.obj) && (r.act == p.act || p.act == "*" ) || r.sub == "root"`) // matchers

	a, err := mongodbadapter.NewAdapterWithCollectionName(options.Client().ApplyURI(lib.config.CasbinConfig.Uri),
		lib.config.CasbinConfig.Database,
		lib.config.CasbinConfig.collectionName) // Your MongoDB URL.
	if err != nil {
		log.Panicf("casbin链接mongod出错 err:%s", err)
	}

	casbinEnforcer, err = casbin.NewEnforcer(m, a)
	if err != nil {
		log.Panicf("初始化casbin失败err:%s", err)
	}

}

// PolicyChange 权限变更
func PolicyChange(userName, path, methods string, add bool) error {
	if add {
		// 先判断权限是否存在
		if casbinEnforcer.HasPolicy(userName, path, methods) {
			return policyExists
		}
		success, err := casbinEnforcer.AddPolicy(userName, path, methods)
		if err != nil || success == false {
			return msgLog(fmt.Sprintf("add policy fail -> %s %s %s err:%s", userName, path, methods, err))
		}
		return nil
	}
	success, err := casbinEnforcer.RemovePolicy(userName, path, methods)
	if err != nil || success == false {
		return msgLog(fmt.Sprintf("remove policy fail -> %s %s %s err:%s", userName, path, methods, err))
	}
	return nil
}

// 获取所有模型权限列表
func (lib *SpAdmin) getModelPolicyList() map[string][]string {
	var result = map[string][]string{}
	// 创建模型权限
	for _, modelList := range lib.mab.GetModelInfoList() {
		result[modelList.MapName] = defaultModelRole
	}
	return result
}

// 获取所有操作权限 返回用户权限列表 模型权限列表 站点权限列表
func (lib *SpAdmin) getAllPolicyList() ([]string, map[string][]string, []string) {
	return defaultUserRole, lib.getModelPolicyList(), defaultSiteRole
}

// 通过resp的方式返回权限列表 适配于前端显示方便
func (lib *SpAdmin) getAllPolicyListResp() permissionsResp {
	var result permissionsResp
	var user = make([]permissionsRespItem, 0, len(defaultUserRole))
	var site = make([]permissionsRespItem, 0, len(defaultSiteRole))
	for _, s := range defaultUserRole {
		user = append(user, permissionsRespItem{
			Title: s,
			Alias: defaultModelRoleAliasMap[s],
			Key:   "mab_user-" + s,
		})
	}
	for _, s := range defaultSiteRole {
		site = append(site, permissionsRespItem{
			Title: s,
			Alias: defaultSiteRoleAliasMap[s],
			Key:   "site-" + s,
		})
	}

	modelPolicyList := lib.getModelPolicyList()

	var models = make([]permissionsRespItem, 0, len(modelPolicyList)*len(defaultModelRole))

	for k, v := range modelPolicyList {

		var inline = make([]permissionsRespItem, 0, len(defaultModelRole))
		for _, s := range v {
			inline = append(inline, permissionsRespItem{
				Title: s,
				Alias: defaultModelRoleAliasMap[s],
				Key:   k + "-" + s,
			})
		}
		var alias string
		var group string
		for _, modelList := range lib.mab.GetModelInfoList() {
			if k == modelList.MapName {
				if len(modelList.Alias) > 0 {
					alias = modelList.Alias
					group = modelList.Group
				}
				break
			}
		}
		models = append(models, permissionsRespItem{
			Title:    k,
			Alias:    alias,
			Key:      k + "_i",
			V:        k,
			Children: inline,
			Group:    group,
		})
	}

	result.Data = append(result.Data, permissionsRespItem{
		Title:    "用户权限",
		Alias:    "用户权限",
		Children: user,
		Key:      defaultModelPolicyName + "_i",
		V:        defaultModelPolicyName,
	})
	result.Data = append(result.Data, permissionsRespItem{
		Title:    "站点权限",
		Alias:    "站点权限",
		Children: site,
		Key:      "site_i",
		V:        "site",
	})
	result.Data = append(result.Data, permissionsRespItem{
		Title:    "数据权限",
		Alias:    "数据权限",
		Children: models,
		Key:      "model_i",
		V:        "model",
	})

	return result
}

// 权限返回resp
func (lib *SpAdmin) policyNeedResp(policys [][]string) permissionsResp {
	var result permissionsResp

	var keys = make(map[string][]string, 0)
	for _, policy := range policys {
		scope := policy[1]
		action := policy[2]
		if v, ok := keys[scope]; ok {
			v = append(v, action)
			keys[scope] = v
		} else {
			keys[scope] = []string{action}
		}
	}

	// 先遍历出 user site
	for k, v := range keys {
		if k == "mab_user" || k == "site" {
			var title string
			switch k {
			case defaultModelPolicyName:
				title = "用户权限"
				break
			case "site":
				title = "站点权限"
				break
			}
			var children = make([]permissionsRespItem, 0, len(v))
			for _, s := range v {
				children = append(children, permissionsRespItem{
					Title: s,
					Key:   k + "-" + s,
				})
			}
			result.Data = append(result.Data, permissionsRespItem{
				Title:    title,
				Alias:    title,
				Key:      k + "_i",
				Children: children,
			})
		}
	}

	var models = make([]permissionsRespItem, 0)
	// 再遍历model
	for k, v := range keys {
		if k != "mab_user" && k != "site" {
			var children = make([]permissionsRespItem, 0, len(v))
			for _, s := range v {
				children = append(children, permissionsRespItem{
					Title: s,
					Key:   k + "-" + s,
				})
			}
			var alias string
			for _, modelList := range lib.mab.GetModelInfoList() {
				if k == modelList.MapName {
					if len(modelList.Alias) > 0 {
						alias = modelList.Alias
					}
					break
				}
			}
			models = append(models, permissionsRespItem{
				Alias:    alias,
				Title:    k,
				Key:      k + "_i",
				Children: children,
			})
		}
	}

	if len(models) > 0 {

		result.Data = append(result.Data, permissionsRespItem{
			Title:    "数据权限",
			Children: models,
			Key:      "model_i",
			V:        "model",
		})
	}

	return result
}
