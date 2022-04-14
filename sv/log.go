package sv

import (
	"log"
	"os"
)

var (
	Warning *log.Logger
)

func init() {
	Warning = log.New(os.Stdout, "[sv] Warning:", log.Ldate|log.Ltime|log.Lshortfile)
}
