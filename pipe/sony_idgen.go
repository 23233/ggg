package pipe

import (
	"fmt"
	"github.com/23233/ggg/ut"
	"github.com/shockerli/cvt"
	"github.com/sony/sonyflake"
	"os"
)

var sf *sonyflake.Sonyflake

// 获取机器ID，这里使用进程ID作为机器ID
func getMachineID() (uint16, error) {
	// 使用进程ID来区分不同进程
	pid := os.Getpid()
	// 通过进程ID生成一个 16 位的机器 ID
	// 你可以根据需要修改生成机器ID的逻辑
	return uint16(pid & 0xFFFF), nil
}

func SfNextId() string {
	uid, err := sf.NextID()
	if err != nil {
		return ut.RandomStr(14)
	}
	s, err := cvt.StringE(uid)
	if err != nil {
		return ut.RandomStr(14)
	}
	return s
}

func InitSf(machineId uint16) *sonyflake.Sonyflake {
	// 使用不同的机器ID初始化 sonyflake
	var st sonyflake.Settings
	st.MachineID = func() (uint16, error) {
		return machineId, nil
	}

	sf = sonyflake.NewSonyflake(st)
	if sf == nil {
		panic("sonyflake not created")
	}
	return sf
}

func init() {
	machineID, err := getMachineID()
	if err != nil {
		panic(fmt.Sprintf("failed to get machine ID: %v", err))
	}
	InitSf(machineID)

}
