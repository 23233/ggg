package pmb

import (
	"context"
	"github.com/23233/ggg/logger"
	"github.com/23233/ggg/pipe"
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/kataras/realip"
	"github.com/qiniu/qmgo"
)

// 记录任何的操作日志
// 需要组合成为 谁在什么事件对谁做了什么
// 暂定记录 新增 修改 删除

var (
	RecordOperationLog = true
)

func ChangeRecordOperationLog(b bool) {
	RecordOperationLog = b
}

type OperationLog struct {
	pipe.ModelBase `bson:",inline"`

	Method   string   `json:"method,omitempty" bson:"method,omitempty" comment:"操作方法"`
	UserId   string   `json:"user_id,omitempty" bson:"user_id,omitempty" comment:"操作用户ID"`
	UserName string   `json:"user_name,omitempty" bson:"user_name,omitempty" comment:"操作用户名"`
	ToSheet  string   `json:"to_sheet,omitempty" bson:"to_sheet,omitempty" comment:"操作的表名"`
	ToRowId  string   `json:"to_row_id,omitempty" bson:"to_row_id,omitempty" comment:"行ID"`
	ToFields []ut.Kov `json:"to_fields,omitempty" bson:"to_fields,omitempty" comment:"字段内容"`
	Msg      string   `json:"msg,omitempty" bson:"msg,omitempty" comment:"消息"`
}

func MustOpLog(ctx iris.Context, db *qmgo.Collection, method string, user *SimpleUserModel, sheet string, msg string, rowId string, toFields []ut.Kov) {
	if !RecordOperationLog {
		return
	}
	var inst = new(OperationLog)
	inst.Method = method
	if user != nil {
		inst.UserId = user.Uid
		inst.UserName = user.NickName
	}
	if user == nil {
		user = &SimpleUserModel{}
		user.Uid = ""
	}

	if toFields == nil {
		toFields = make([]ut.Kov, 0)
	}
	// 加入操作者设备信息
	toFields = append(toFields, ut.Kov{
		Key:   "ua",
		Value: ctx.GetHeader("User-Agent"),
	})
	toFields = append(toFields, ut.Kov{
		Key:   "ip",
		Value: realip.Get(ctx.Request()),
	})

	inst.ToSheet = sheet
	inst.ToRowId = rowId
	inst.ToFields = toFields
	inst.Msg = msg
	_ = inst.BeforeInsert(context.TODO())

	_, err := db.InsertOne(context.TODO(), inst)
	if err != nil {
		logger.J.ErrorE(err, "%s进行 %s-%s 操作记录失败", user.Uid, method, sheet)
	}
}

func OpLogSyncIndex(ctx context.Context, coll *qmgo.Collection) error {
	cl, err := coll.CloneCollection()
	if err != nil {
		return err
	}
	err = ut.MCreateIndex(ctx, cl,
		ut.MGenNormal("user_id"),
		ut.MGenNormal("user_name"),
		ut.MGenNormal("to_sheet"),
		ut.MGenNormal("to_row_id"),
	)
	return err
}
