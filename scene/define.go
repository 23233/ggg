package scene

import (
	"github.com/23233/ggg/ut"
)

type ContextValueInject struct {
	FromKey    string `json:"from_key,omitempty"`
	ToKey      string `json:"to_key,omitempty"`
	AllowEmpty bool   `json:"allow_empty,omitempty"`
}

type SchemaGetResp struct {
	*ut.MongoFacetResult
	Page     int64          `json:"page,omitempty"`
	PageSize int64          `json:"page_size,omitempty"`
	Sorts    *ut.BaseSort   `json:"sorts,omitempty"`
	Filters  *ut.QueryParse `json:"filters,omitempty"`
}

type RequestModule struct {
	Scope  string `json:"scope"`  // 作用域 例如说 miniapp admin
	Model  string `json:"model"`  // 对应的模型 tasks
	Scene  string `json:"scene"`  // 对应的场景
	Action string `json:"action"` // 对应的操作 getAllModel
}

type RequestData struct {
	Module RequestModule     `json:"module"`
	Query  map[string]string `json:"query"` // 查询参数 跟解析url参数一样
	Body   string            `json:"body"`  // 这里是json字符串 需要序列化
	Comm   map[string]string `json:"comm"`  // 这里放置一些通用信息 版本号之类的 map即可
}

func (c *RequestData) GetUid() string {
	uid, _ := c.Query[ut.DefaultUidTag]
	return uid
}

type ModelInfo struct {
	Group     string `json:"group,omitempty"`      // 组名
	Priority  int    `json:"priority,omitempty"`   // 在组下显示的优先级 越大越优先
	TableName string `json:"table_name,omitempty"` // 表名
	UniqueId  string `json:"unique_id,omitempty"`  // 唯一ID 默认生成sonyflakeId
	PathId    string `json:"path_id,omitempty"`    // 路径ID 默认取UniqueId 在加入backend时会自动修改
	RawName   string `json:"raw_name,omitempty"`   // 原始名称
	Alias     string `json:"alias,omitempty"`      // 别名 中文名
}
