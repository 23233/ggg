package contentSafe

import (
	"embed"
	"github.com/23233/ggg/logger"
	"github.com/23233/lad"
)

//go:embed *.txt
var words embed.FS

var LadClient = lad.New()

func InitClient() error {
	err := LadClient.LoadOfFolder(words)
	if err != nil {
		return err
	}
	LadClient.Build()
	return nil
}

func init() {
	err := InitClient()
	if err != nil {
		logger.J.Error("初始化lad失败", logger.J.WithError(err))
	}
}
