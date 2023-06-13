package logger

import "go.uber.org/zap"

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
	p := "j_"
	if len(prefix) >= 1 {
		p = prefix
	}
	jt := New()
	jt.SetEnableStats(true)
	jt.SetEnableQueue(true) // 启动错误队列
	jt.SetDivision("time")
	jt.SetEncoding("json")                     // 输出格式 "json" 或者 "console"
	jt.SetTimeUnit(t)                          // 按天归档
	jt.SetInfoFile(DefaultPath + p + "info")   // 设置info级别日志
	jt.SetErrorFile(DefaultPath + p + "error") // 设置error级别日志
	jt.SetCaller(true)
	jt.SetCallerSkip(1)
	jt.Fields = fields
	return jt.InitLogger()
}

func InitJsonSizeLog(prefix string, fields ...zap.Field) *Log {
	p := "s_"
	if len(prefix) >= 1 {
		p = prefix
	}
	js := New()
	js.SetEnableStats(true)
	js.SetEnableQueue(true)
	js.SetDivision("size")
	js.SetEncoding("json")
	js.SetInfoFile(DefaultPath + p + "s_info")   // 设置info级别日志
	js.SetErrorFile(DefaultPath + p + "s_error") // 设置error级别日志
	js.MaxSize = 500
	js.MaxAge = 28
	js.Compress = true
	js.MaxBackups = 10
	js.SetCaller(true)
	js.SetCallerSkip(1)
	js.Fields = fields
	return js.InitLogger()
}

func init() {
	J = InitJsonTimeLog("", Day)
	Js = InitJsonSizeLog("")
	JH = InitJsonTimeLog("", Hour)
}
