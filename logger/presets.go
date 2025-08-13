package logger

import (
	"fmt"
	"go.uber.org/zap"
	"os" // 1. 导入 "os" 包
	"time"
)

var (
	DefaultPath = "./logs/"
	J           *Log // json day
	JH          *Log // json hour
	Js          *Log // json size
	JM          *Log // json day size mix
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
	jt.SetEncoding("json")     // 输出格式 "json" 或者 "console"
	jt.SetTimeUnit(t)          // 按天归档
	jt.SetInfoFile(infoPath)   // 设置info级别日志
	jt.SetErrorFile(errorPath) // 设置error级别日志
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

func InitJsonTimeSizeLog(prefix string, t TimeUnit, maxSize int, fields ...zap.Field) *Log {
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
	jts.MaxSize = maxSize
	jts.MaxAge = 28 // Default to 28 days
	jts.Compress = false
	jts.MaxBackups = 10
	jts.SetCaller(true)
	jts.SetCallerSkip(1)
	jts.Fields = fields
	return jts.InitLogger()
}

func init() {
	// 2. 检查DefaultPath是否存在
	// os.Stat返回文件信息，如果路径不存在，它会返回一个错误
	if _, err := os.Stat(DefaultPath); os.IsNotExist(err) {
		// 3. 如果路径不存在，则创建它
		// os.MkdirAll会创建路径中的所有父目录（如果需要的话）
		// 0755是目录权限，表示所有者有读/写/执行权限，组和其他用户有读/执行权限
		err = os.MkdirAll(DefaultPath, 0755)
		if err != nil {
			fmt.Println(fmt.Sprintf("创建日志目录失败: %v", err))
		}
	}

	J = InitJsonTimeLog("", Day)
	Js = InitJsonSizeLog("")
	JH = InitJsonTimeLog("", Hour)
	JM = InitJsonTimeSizeLog("", Day, 50)
}
