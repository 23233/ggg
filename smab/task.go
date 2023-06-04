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
	Name   string `json:"name,omitempty" bson:"name,omitempty"`       // 不能重复
	ReqUri string `json:"req_uri,omitempty" bson:"req_uri,omitempty"` // 操作请求地址
	Scheme string `json:"scheme,omitempty" bson:"scheme,omitempty" `  // 需要用户填写的表单数据
}

// SmTask 任务
type SmTask struct {
	DefaultField `bson:",inline,flatten"`
	Name         string             `json:"name,omitempty" bson:"name,omitempty" comment:"任务名称"`
	Desc         string             `json:"desc,omitempty" bson:"desc,omitempty" comment:"任务描述"`
	Type         uint8              `json:"type,omitempty" bson:"type,omitempty" comment:"任务类型"` // 任务类型
	Group        string             `json:"group,omitempty" bson:"group,omitempty" comment:"任务组"`
	Tags         []string           `json:"tags,omitempty" bson:"tags,omitempty" comment:"标签组"`
	Content      string             `json:"content,omitempty" bson:"content,omitempty" comment:"任务内容" mab:"t=markdown"` // 任务内容 markdown格式
	InjectData   string             `json:"inject_data,omitempty" bson:"inject_data,omitempty" comment:"t=textarea"`    // 任务注入的json字符串 可以自行序列化回去
	Action       []ActionItem       `json:"action,omitempty" bson:"action,omitempty" comment:"按钮组"`
	ExpTime      time.Time          `json:"exp_time,omitempty" bson:"exp_time,omitempty" comment:"任务过期时间"`       // 任务过期时间
	ToUser       primitive.ObjectID `json:"to_user,omitempty" bson:"to_user,omitempty" comment:"操作的用户"`          // 展示的用户
	CreateUser   primitive.ObjectID `json:"create_user,omitempty" bson:"create_user,omitempty" comment:"创建用户"`   // 创建的用户
	Success      bool               `json:"success,omitempty" bson:"success,omitempty" comment:"操作完成?"`          // 操作完成
	Msg          string             `json:"msg,omitempty" bson:"msg,omitempty" comment:"操作结果"`                   // 操作结果
	AllowDelete  bool               `json:"allow_delete,omitempty" bson:"allow_delete,omitempty" comment:"允许删除"` // 是否允许删除
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
	task.InjectData = injectStrData
	task.Action = PassOrRejectAction(postUrl)
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
