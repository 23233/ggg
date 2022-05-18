package smab

import (
	"github.com/kataras/iris/v12"
	"path/filepath"
	"runtime/debug"
)

type CasbinConfigDefine struct {
	Uri            string
	Database       string
	collectionName string
}

type WelComeConfigDefine struct {
	Title    string   `json:"title"`     // 标题
	MainText []string `json:"main_text"` //
	Desc     []string `json:"desc"`      //
}

type Configs struct {
	Name             string
	App              *iris.Application
	ModelList        []interface{}          // 模型列表
	AbridgeName      string                 // tag的解析名称
	Prefix           string                 // 前缀
	AllowTokenLogin  bool                   // 是否允许root使用token登录
	OnFileUpload     func(ctx iris.Context) // 图片上传事件 成功返回JSON{origin:"",thumbnail:""} origin必须存在 失败则JSON{detail:"失败理由"}
	PublicKey        string                 // 公开密钥 如果有密钥则会对上传进行介入
	CasbinConfig     CasbinConfigDefine
	GlobalQianKun    []QianKunConfigExtra // 全局所有用户都能看到的前端信息
	SuperUserQianKun []QianKunConfigExtra // 仅管理员可见
	WelComeConfig    WelComeConfigDefine
}

func (c *Configs) initConfig() Configs {
	return Configs{
		Name:        "管理后台",
		AbridgeName: "sp",
		Prefix:      "/admin",
	}
}

func (c *Configs) valid() error {
	if c.App == nil {
		return msgLog("app是必须的")
	}
	if len(c.Prefix) < 1 {
		return msgLog("前缀prefix是必须的")
	}
	if c.Prefix[0] != '/' {
		c.Prefix = "/" + c.Prefix
	}
	if len(c.CasbinConfig.Uri) < 1 {
		return msgLog("权限mongouri是必须设置的")
	}
	if len(c.CasbinConfig.Database) < 1 {
		// get module info
		bi, ok := debug.ReadBuildInfo()
		if !ok {
			c.CasbinConfig.Database = "casbin"
		} else {
			// if you go.mod module is example.com/foo --> foo_casbin
			// if you go.mod module is zoo --> zoo_casbin
			c.CasbinConfig.Database = filepath.Base(bi.Main.Path) + "_casbin"
		}
	}
	if len(c.CasbinConfig.collectionName) < 1 {
		c.CasbinConfig.collectionName = "casbin_rule"
	}
	return nil
}
