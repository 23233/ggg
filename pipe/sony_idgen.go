package pipe

import (
	"github.com/23233/ggg/ut"
	"github.com/shockerli/cvt"
	"github.com/sony/sonyflake"
)

var sf *sonyflake.Sonyflake

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

func init() {
	var st sonyflake.Settings
	sf = sonyflake.NewSonyflake(st)
	if sf == nil {
		panic("sonyflake not created")
	}
}
