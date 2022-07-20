package smab

import (
	"context"
	"testing"
)

func TestCreateTask(t *testing.T) {

	var m = GenTaskAtRoot("测试任务", "任务描述")
	m.Group = "赵日天"
	m.Type = 10
	m.InjectData = `{"obj_id":"98439834"}`
	m.Action = PassOrNotReasonAction("/test", "/test")
	m.Content = `### 我是赵日天 \r 我今天贼开心 你开不开心啊`
	err := CreateTask(context.Background(), m)
	if err != nil {
		t.Fatal(err)
	}

}

func TestGenTaskInjectData(t *testing.T) {

	var m = map[string]interface{}{
		"field1": "string",
		"field2": 123,
	}

	result := GenTaskInjectData(m)
	if len(result) < 1 {
		t.Fatal("map解构出错")
	}

	type testReq struct {
		AndOne string `json:"and_one"`
		Two    string `json:"two"`
		Three  int    `json:"three"`
	}

	var st testReq
	st.AndOne = "1"
	st.Two = "2"
	st.Three = 3
	result = GenTaskInjectData(st)
	if len(result) < 1 {
		t.Fatal("struct解构出错")
	}

}
