package logger

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

	c.SetEnableQueue(true)

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

	t.Logf("info queue size : %d", c.InfoQueue().Size())
	t.Logf("error queue size : %d", c.ErrorQueue().Size())
	t.Logf("info quque list :%v", c.InfoQueue().ItemsMap())
	t.Logf("error quque list :%v", c.ErrorQueue().ItemsMap())

}

func TestViewQueueFunc(t *testing.T) {
	c := New()
	c.SetEnableQueue(true)
	c.SetEncoding("json") // 输出格式 "json" 或者 "console"

	c.SetInfoFile("./logs/server.log")      // 设置info级别日志
	c.SetErrorFile("./logs/server_err.log") // 设置warn级别日志

	c.InitLogger()

	Info("info level test")
	Error("dsdadadad level test", WithError(errors.New("sabhksasas")))
	Error("121212121212 error")
	Warn("warn level test")
	Debug("debug level test")

	t.Logf("info queue size : %d", c.InfoQueue().Size())
	t.Logf("error queue size : %d", c.ErrorQueue().Size())

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	ViewQueueFunc(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	if !strings.Contains(rr.Body.String(), "info log list view") {
		t.Error(errors.New("not get success data"))
	}

	//t.Log("open browser view is success http://127.0.0.1:8787 ")
	//http.HandleFunc("/", ViewQueueFunc)
	//http.ListenAndServe(":8787", nil)

}

func TestViewStatsFunc(t *testing.T) {
	c := New()
	c.SetEncoding("json") // 输出格式 "json" 或者 "console"
	c.SetEnableStats(true)
	c.SetStatsFormat("2006-01-02 15:04:05")

	c.SetInfoFile("./logs/server.log")      // 设置info级别日志
	c.SetErrorFile("./logs/server_err.log") // 设置warn级别日志

	c.InitLogger()

	for i := 1; i < 10; i++ {
		for ii := 0; ii < 100-(i*10); ii++ {
			Warn("warn level test")
		}
		time.Sleep(2 * time.Second)
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	ViewStatsFunc(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	//
	//t.Log("open browser view is success http://127.0.0.1:8787 ")
	//http.HandleFunc("/", ViewStatsFunc)
	//http.ListenAndServe(":8787", nil)

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
