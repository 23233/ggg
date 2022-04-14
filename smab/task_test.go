package smab

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
)

func TestCreateTask(t *testing.T) {

	var m = GenTask("测试任务", "任务描述")
	m.Group = "赵日天"
	m.Type = 10
	m.CreateUser = primitive.NewObjectID()
	m.ToUser, _ = primitive.ObjectIDFromHex("60de74ee4c43d1fa93369ba5")
	m.Action = PassOrNotReasonAction("/test", "/test", `{"obj_id":"98439834"}`)
	m.Content = `### 我是赵日天 \r 我今天贼开心 你开不开心啊`
	err := CreateTask(context.Background(), m)
	if err != nil {
		t.Fatal(err)
	}

}
