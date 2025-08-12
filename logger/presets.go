package logger

import (
	"fmt"
	"go.uber.org/zap"
	"time"
)

var (
	DefaultPath = "./logs/"
	J           *Log
	JH          *Log
	Js          *Log
)

func ChangeJtCaller(call int) {
	J.Op.SetCallerSkip(call)
	J = J.Op.InitLogger()
}
func ChangeJsCaller(call int) {
	Js.Op.SetCallerSkip(call)
	Js = Js.Op.InitLogger()
}

func InitJsonTimeLog(prefix string, t TimeUnit, fields ...zap.Field) *Log {
	instanceName := prefix
	if instanceName == "" {
		instanceName = time.Now().Format("20060102150405")
	}
	infoPath := fmt.Sprintf("%sj_%s_info", DefaultPath, instanceName)
	errorPath := fmt.Sprintf("%sj_%s_error", DefaultPath, instanceName)

	jt := New()
	jt.SetEnableStats(true)
	jt.SetEnableQueue(true) // 启动错误队列
	jt.SetDivision("time")
	jt.SetEncoding("json")                     // 输出格式 "json" 或者 "console"
	jt.SetTimeUnit(t)                          // 按天归档
	jt.SetInfoFile(infoPath)                   // 设置info级别日志
	jt.SetErrorFile(errorPath)                 // 设置error级别日志
	jt.SetCaller(true)
	jt.SetCallerSkip(1)
	jt.Fields = fields
	return jt.InitLogger()
}

func InitJsonSizeLog(prefix string, fields ...zap.Field) *Log {
	instanceName := prefix
	if instanceName == "" {
		instanceName = time.Now().Format("20060102150405")
	}
	infoPath := fmt.Sprintf("%ss_%s_info", DefaultPath, instanceName)
	errorPath := fmt.Sprintf("%ss_%s_error", DefaultPath, instanceName)
	js := New()
	js.SetEnableStats(true)
	js.SetEnableQueue(true)
	js.SetDivision("size")
	js.SetEncoding("json")
	js.SetInfoFile(infoPath)
	js.SetErrorFile(errorPath)
	js.MaxSize = 500
	js.MaxAge = 28
	js.Compress = true
	js.MaxBackups = 10
	js.SetCaller(true)
	js.SetCallerSkip(1)
	js.Fields = fields
	return js.InitLogger()
}

func InitJsonTimeSizeLog(prefix string, t TimeUnit, fields ...zap.Field) *Log {
	instanceName := prefix
	if instanceName == "" {
		instanceName = time.Now().Format("20060102150405")
	}
	infoPath := fmt.Sprintf("%sts_%s_info", DefaultPath, instanceName)
	errorPath := fmt.Sprintf("%sts_%s_error", DefaultPath, instanceName)

	jts := New()
	jts.SetEnableStats(true)
	jts.SetEnableQueue(true)
	jts.SetDivision(TimeAndSizeDivision)
	jts.SetEncoding("json")
	jts.SetTimeUnit(t)
	jts.SetInfoFile(infoPath)
	jts.SetErrorFile(errorPath)
	jts.MaxSize = 500 // Default to 500MB
	jts.MaxAge = 28    // Default to 28 days
	jts.Compress = true
	jts.MaxBackups = 10
	jts.SetCaller(true)
	jts.SetCallerSkip(1)
	jts.Fields = fields
	return jts.InitLogger()
}

func init() {
	J = InitJsonTimeLog("", Day)
	Js = InitJsonSizeLog("")
	JH = InitJsonTimeLog("", Hour)
}
