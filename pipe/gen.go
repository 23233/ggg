package pipe

import (
	"github.com/23233/ggg/ut"
	"github.com/pkg/errors"
)

// Schema
// type : string number(默认float64) integer(默认 uint64) object array boolean null
// format 约定列表
// 外键 fk:表名,字段名(默认uid) eg:  fk:dsfjsidjf  fk:dsjfiwijefi,uid
// markdown
// image
// widget 约定
// 字段编辑器 rawSchema

type Attribute struct {
	GenType   string `json:"gen_type,omitempty" bson:"gen_type,omitempty"`     // 自动生成方式
	GenLen    int    `json:"gen_len,omitempty" bson:"gen_len,omitempty"`       // 生成长度
	GenStat   int    `json:"gen_stat,omitempty" bson:"gen_stat,omitempty"`     // 自动生成起始值
	GenEnd    int    `json:"gen_end,omitempty" bson:"gen_end,omitempty"`       // 自动生成结束值
	AllowEdit bool   `json:"allow_edit,omitempty" bson:"allow_edit,omitempty"` // 生成的值是否允许修改 默认不允许
}

func (c *Attribute) RunGen() (any, error) {
	switch c.GenType {
	case "uuid":
		return GenUUid(), nil
	case "sfid":
		return SfNextId(), nil
	case "string":
		if c.GenLen < 1 {
			return nil, errors.New("生成参数错误")
		}
		return ut.RandomStr(c.GenLen), nil
	case "int":
		if c.GenEnd < c.GenStat || c.GenStat < 1 {
			return nil, errors.New("生成参数错误")
		}
		return ut.RandomInt(c.GenStat, c.GenEnd), nil
	}
	return nil, errors.New("未配置生成规则")
}
