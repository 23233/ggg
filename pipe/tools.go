package pipe

import (
	uuid "github.com/iris-contrib/go.uuid"
	"strings"
)

func GenUUid() string {
	uidV4, _ := uuid.NewV4()
	OutTradeNo := strings.ReplaceAll(uidV4.String(), "-", "")
	return OutTradeNo
}
