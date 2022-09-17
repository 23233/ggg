package logger

var (
	DefaultPath = "./logs/"
	J           *Log
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

func initJsonTimeLog() {
	jt := New()
	jt.SetEnableStats(true)
	jt.SetEnableQueue(true) // 启动错误队列
	jt.SetDivision("time")
	jt.SetEncoding("json")                       // 输出格式 "json" 或者 "console"
	jt.SetTimeUnit(Day)                          // 按天归档
	jt.SetInfoFile(DefaultPath + "j_info.log")   // 设置info级别日志
	jt.SetErrorFile(DefaultPath + "j_error.log") // 设置error级别日志
	jt.SetCaller(true)
	jt.SetCallerSkip(1)
	J = jt.InitLogger()
}

func initJsonSizeLog() {
	js := New()
	js.SetEnableStats(true)
	js.SetEnableQueue(true)
	js.SetDivision("size")
	js.SetEncoding("json")
	js.SetInfoFile(DefaultPath + "s_info.log")   // 设置info级别日志
	js.SetErrorFile(DefaultPath + "s_error.log") // 设置error级别日志
	js.MaxSize = 500
	js.MaxAge = 28
	js.Compress = true
	js.MaxBackups = 10
	js.SetCaller(true)
	js.SetCallerSkip(1)
	Js = js.InitLogger()
}

func init() {
	initJsonTimeLog()
	initJsonSizeLog()
}
