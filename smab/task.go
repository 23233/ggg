package smab

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// ActionItem 操作
type ActionItem struct {
	Name   string `json:"name" bson:"name"`       // 不能重复
	ReqUri string `json:"req_uri" bson:"req_uri"` // 操作请求地址
	Built  string `json:"built" bson:"built"`     // 内置数据 json string
	Scheme string `json:"scheme" bson:"scheme" `  // 需要用户填写的表单数据
}

// SmTask 任务
type SmTask struct {
	DefaultField `bson:",inline,flatten"`
	Name         string             `json:"name" bson:"name" comment:"任务名称"`
	Desc         string             `json:"desc" bson:"desc" comment:"任务描述"`
	Type         uint8              `json:"type" bson:"type" comment:"任务类型"` // 任务类型
	Group        string             `json:"group" bson:"group" comment:"任务组"`
	Content      string             `json:"content" bson:"content" comment:"任务内容" mab:"t=markdown"` // 任务内容 markdown格式
	Action       []ActionItem       `json:"action" bson:"action" comment:"按钮组"`
	ExpTime      time.Time          `json:"exp_time" bson:"exp_time" comment:"任务过期时间"`       // 任务过期时间
	ToUser       primitive.ObjectID `json:"to_user" bson:"to_user" comment:"操作的用户"`          // 展示的用户
	CreateUser   primitive.ObjectID `json:"create_user" bson:"create_user" comment:"创建用户"`   // 创建的用户
	Success      bool               `json:"success" bson:"success" comment:"操作完成?"`          // 操作完成
	Msg          string             `json:"msg" bson:"msg" comment:"操作结果"`                   // 操作结果
	AllowDelete  bool               `json:"allow_delete" bson:"allow_delete" comment:"允许删除"` // 是否允许删除
}

// CreateTask 创建任务
func CreateTask(ctx context.Context, t *SmTask) error {
	one, err := getCollName("sm_task").InsertOne(ctx, t)
	if err != nil {
		return err
	}
	if one.InsertedID.(primitive.ObjectID).IsZero() {
		return errors.New("新增失败")
	}
	return nil
}

// DeleteTask 删除任务
func DeleteTask(ctx context.Context, id string) error {
	objId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	err = getCollName("sm_task").RemoveId(ctx, objId)
	if err != nil {
		return err
	}
	return nil
}

// SetTaskSuccess 设置任务完成
func SetTaskSuccess(ctx context.Context, id string, success bool, msg string) error {
	objId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	err = getCollName("sm_task").UpdateId(ctx, objId, bson.M{"$set": bson.M{"success": success, "msg": msg}})
	if err != nil {
		return err
	}
	return nil
}

// GetTask 获取任务
func GetTask(ctx context.Context, id string) (*SmTask, error) {
	objId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var r = new(SmTask)
	err = getCollName("sm_task").Find(ctx, bson.M{"_id": objId}).One(&r)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// GenTask 生成任务
func GenTask(name string, desc ...string) *SmTask {
	var d string
	if len(desc) >= 1 {
		d = desc[0]
	}
	return &SmTask{
		Name: name,
		Desc: d,
	}
}

// GenTaskAtRoot 生成root任务
func GenTaskAtRoot(name string, desc ...string) *SmTask {
	var t = GenTask(name, desc...)
	t.ToUser = RootUser.Id
	t.CreateUser = RootUser.Id
	return t
}

// GenTaskGroupAtRoot 生成特定组的root任务
func GenTaskGroupAtRoot(name string, group string) *SmTask {
	var t = GenTaskAtRoot(name)
	t.Group = group
	return t
}

// GenMarkdownVerifyTask 快速创建一个基于markdown介绍的审核类任务
func GenMarkdownVerifyTask(ctx context.Context, name string, group string, desc string, content TaskMarkdownState, postUrl string, injectStrData string) error {
	task := GenTaskGroupAtRoot(name, group)
	task.Desc = desc
	task.Content = content.GetStr()
	task.Action = PassOrRejectAction(postUrl, injectStrData)
	return CreateTask(ctx, task)
}

// GenTaskInjectData 快速生成inject的数据 可以是struct 也可以是map
func GenTaskInjectData(input any) string {
	jsonStr, err := json.Marshal(&input)
	if err == nil {
		return string(jsonStr)
	}
	return ""
}
