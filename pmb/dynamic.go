package pmb

import (
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"go.mongodb.org/mongo-driver/bson"
)

type DynamicResult struct {
	ShowType string `json:"show_type"`
	Data     any    `json:"data"`
}

func (r *DynamicResult) Normal(data string) {
	r.ShowType = "normal"
	r.Data = data
}
func (r *DynamicResult) Raw(data any) {
	r.ShowType = "raw"
	r.Data = data
}

type DynamicCall func(fieldId string, model IModelItem, user *SimpleUserModel, row map[string]any) (*DynamicResult, error)
type DynamicField struct {
	Id      string `json:"id"`      // 字段唯一id
	Name    string `json:"name"`    // 字段名
	Trigger string `json:"trigger"` // 触发方式 默认interval
	call    DynamicCall
}

func (c *DynamicField) AddCall(call DynamicCall) *DynamicField {
	c.call = call
	return c
}
func (c *DynamicField) SetTrigger(trigger string) *DynamicField {
	c.Trigger = trigger
	return c
}
func (c *DynamicField) SetTriggerInterval() *DynamicField {
	c.Trigger = "interval"
	return c
}
func (c *DynamicField) SetTriggerClick() *DynamicField {
	c.Trigger = "click"
	return c
}

func NewDynamicField(id, name string) *DynamicField {
	return &DynamicField{
		Id:   id,
		Name: name,
	}
}

type DynamicReq struct {
	RowUid string `json:"row_uid"` // 行id
	Id     string `json:"id"`      // 动态字段id
}

func DynamicRun(ctx iris.Context, model IModelItem, user *SimpleUserModel) {
	part := new(DynamicReq)
	err := ctx.ReadBody(&part)
	if err != nil {
		IrisRespErr("解析字段参数包失败", err, ctx)
		return
	}
	if len(part.RowUid) < 1 || len(part.Id) < 1 {
		IrisRespErr("参数错误", err, ctx)
		return
	}
	df := model.DfIdFind(part.Id)
	if df == nil {
		IrisRespErr("获取动态字段匹配项失败", err, ctx)
		return
	}

	// 获取这一行的数据
	var row = make(map[string]any)
	err = model.GetCollection().Find(ctx, bson.M{ut.DefaultUidTag: part.RowUid}).One(&row)
	if err != nil {
		IrisRespErr("获取对应行数据失败", err, ctx)
		return
	}

	result, err := df.call(part.Id, model, user, row)
	if err != nil {
		IrisRespErr("", err, ctx)
		return
	}
	ctx.JSON(result)

}
