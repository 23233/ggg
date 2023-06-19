package pipe

import (
	"github.com/23233/ggg/ut"
	"os"
	"time"
)

type PipeInfo struct {
	Key  string `json:"key,omitempty"`
	Name string `json:"name,omitempty"`
	Desc string `json:"desc,omitempty"`
}

func NewPipeInfo(name string) *PipeInfo {
	return &PipeInfo{
		Name: name,
	}
}

// PipeRunResp 操作序列执行结果
type PipeRunResp[T any] struct {
	result       T     // 执行结果
	err          error // 错误
	reqCode      int   // 请求状态码 权重在pipeline定义的errCode之后
	businessCode int   // 业务code 仅在错误时有则会返回
	isBreak      bool  // 是否中断之后的执行
}

func (c *PipeRunResp[T]) SetBusinessCode(businessCode int) *PipeRunResp[T] {
	c.businessCode = businessCode
	return c
}
func (c *PipeRunResp[T]) SetReqCode(reqCode int) *PipeRunResp[T] {
	c.reqCode = reqCode
	return c
}
func (c *PipeRunResp[T]) SetBreak(b bool) *PipeRunResp[T] {
	c.isBreak = b
	return c
}

func newPipeErr[T any](err error) *PipeRunResp[T] {
	return &PipeRunResp[T]{
		err: err,
	}
}
func newPipeResult[T any](result T) *PipeRunResp[T] {
	return &PipeRunResp[T]{
		result: result,
	}
}
func newPipeResultErr[T any](result T, err error) *PipeRunResp[T] {
	return &PipeRunResp[T]{
		err:    err,
		result: result,
	}
}

// 以下就是各种pipe的定义了 需要挪到单个文件中去

// ModelCtxMapperArgs 模型上下文映射参数 适用于post 和put 请求
type ModelCtxMapperArgs struct {
	OmitKeys   []string              `json:"omit_keys,omitempty" bson:"omit_keys,omitempty"`     // 需要跳过的keys
	GenKeys    map[string]*Attribute `json:"gen_keys,omitempty" bson:"gen_keys,omitempty"`       // 代码生成的数据
	InjectData map[string]any        `json:"inject_data,omitempty" bson:"inject_data,omitempty"` // 注入的数据
}

type StrTemplate struct {
	VarName string `json:"var_name,omitempty"`
	Value   any    `json:"value,omitempty"`
}

// StrExpand 字符串展开 字符串模板
type StrExpand struct {
	Key    string        `json:"key,omitempty"`
	KeyMap []StrTemplate `json:"key_map,omitempty"`
}

func (c *StrExpand) Build() (string, error) {
	if c == nil {
		return "", nil
	}
	attach := c.Key
	if len(c.KeyMap) > 0 {
		am := make(map[string]string)
		// 注入常量
		am["now_rfc3339"] = time.Now().Format(time.RFC3339)
		am["now"] = time.Now().Format("2006-01-02 15:04:05")
		for _, temp := range c.KeyMap {
			v, err := ut.TypeChange(temp.Value, "string")
			if err != nil {
				return "", err
			}
			vv := v.(string)
			am[temp.VarName] = vv
		}
		attach = os.Expand(attach, func(k string) string {
			v, ok := am[k]
			if !ok {
				return ""
			}
			return v
		})
	}
	return attach, nil
}

// MongoOperateField 会生成 bson.M{<warp>:<children | value | nil>}
// warp $set{}
// https://www.mongodb.com/docs/manual/reference/method/db.collection.updateMany/#mongodb-method-db.collection.updateMany
// https://www.mongodb.com/docs/manual/reference/operator/update/
