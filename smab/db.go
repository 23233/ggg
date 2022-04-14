package smab

import (
	"github.com/qiniu/qmgo"
)

func getCollName(name string) *qmgo.Collection {
	return NowSp.Mdb.Collection(name)
}
