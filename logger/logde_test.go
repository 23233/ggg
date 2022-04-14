package logger

import (
	"errors"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	c := New()
	c.SetDivision("time") // 设置归档方式，"time"时间归档 "size" 文件大小归档，文件大小等可以在配置文件配置
	c.SetTimeUnit(Day)    // 时间归档 可以设置切割单位
	c.SetEncoding("json") // 输出格式 "json" 或者 "console"
	//c.Stacktrace = true
	c.SetCaller(true)
	c.SetCallerSkip(1) // 如果需要自己封装的话 请再次+1 只到调用文件行号能够顺利显示未知

	c.SetInfoFile("./logs/server.log")      // 设置info级别日志
	c.SetErrorFile("./logs/server_err.log") // 设置warn级别日志

	//c.SentryConfig = SentryLoggerConfig{
	//	DSN:              "sentry dsn",
	//	Debug:            true,
	//	AttachStacktrace: true,
	//	Environment:      "dev",
	//	Tags: map[string]string{
	//		"source": "demo",
	//	},
	//}

	c.InitLogger()

	Info("info level test")
	Error("dsdadadad level test", WithError(errors.New("sabhksasas")))
	Error("121212121212 error")
	Warn("warn level test")
	Debug("debug level test")

	time.Sleep(2 * time.Second) // 避免程序结束太快，没有上传sentry

	Info("this is a log", With("trace", "12345677"))
	Info("this is a log", WithError(errors.New("this is a new error")))
}

func BenchmarkLogger(b *testing.B) {
	b.Logf("Logging at a disabled level with some accumulated context.")
	b.Run("logde logger without fields", func(b *testing.B) {
		c := New()
		c.CloseConsoleDisplay()
		c.InitLogger()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				Info("1234")
			}
		})
	})
	b.Run("logde logger with fields", func(b *testing.B) {
		c := New()
		c.CloseConsoleDisplay()
		c.InitLogger()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				Info("1234", With("Trace", "1234455"))
			}
		})
	})
	b.Run("logde logger without fields write into file", func(b *testing.B) {
		c := New()
		c.CloseConsoleDisplay()
		c.SetInfoFile("../logs/test_stdout.log")
		c.InitLogger()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				Info("1234")
			}
		})
	})
	b.Run("logde logger with fields write into file", func(b *testing.B) {
		c := New()
		c.CloseConsoleDisplay()
		c.SetInfoFile("../logs/test_stdout.log")
		c.InitLogger()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				Info("1234", With("Trace", "1234455"))
			}
		})
	})
}
