package pipe

import (
	"github.com/redis/rueidis"
	"golang.org/x/net/context"
	"testing"
)

func TestSmsClient_Send(t *testing.T) {
	rdb, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{"127.0.0.1:6379"},
		Password:    "password",
		SelectDB:    4,
	})
	if err != nil {
		t.Error(err)
	}
	mobile := "13866666666"
	inst := NewDefaultSmsClient(rdb)
	code, err := inst.SendBeforeCheck(context.TODO(), "123456", mobile)
	if err != nil {
		t.Error(err)
		return
	}
	pass := inst.Valid(context.TODO(), mobile, code)
	if !pass {
		t.Error(err)
	}

}
