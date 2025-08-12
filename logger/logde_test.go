package logger

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {

	l := InitJsonTimeLog("t_", Day, zap.Any("ttt", "bbb"))
	//l.Op.AddField("ttt", "bbb")
	l.Info("info level test")
	l.Error("dsdadadad level test", l.WithError(errors.New("sabhksasas")))
	l.Error("121212121212 error")
	l.Warn("warn level test")
	l.Debug("debug level test")
	l.ErrEf(errors.New("dsjiofjwoiejf"), "real %s", "yes")

	time.Sleep(2 * time.Second) // 避免程序结束太快，没有上传sentry

	l.Info("this is a log", l.With("trace", "12345677"))
	l.Info("this is a log", l.WithError(errors.New("this is a new error")))

	t.Logf("info queue size : %d", l.Op.InfoQueue().Size())
	t.Logf("error queue size : %d", l.Op.ErrorQueue().Size())
	t.Logf("info quque list :%v", l.Op.InfoQueue().ItemsMap())
	t.Logf("error quque list :%v", l.Op.ErrorQueue().ItemsMap())

}

func TestViewQueueFunc(t *testing.T) {

	J.Info("info level test")
	J.Error("dsdadadad level test", J.WithError(errors.New("sabhksasas")))
	J.Error("121212121212 error")
	J.Warn("warn level test")
	J.Debug("debug level test")

	t.Logf("info queue size : %d", J.Op.InfoQueue().Size())
	t.Logf("error queue size : %d", J.Op.ErrorQueue().Size())

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	J.ViewQueueFunc(rr, req)

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
	for i := 1; i < 10; i++ {
		for ii := 0; ii < 100-(i*10); ii++ {
			J.Warn("warn level test")
		}
		time.Sleep(2 * time.Second)
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	J.ViewStatsFunc(rr, req)

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
				J.Info("1234")
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
				J.Info("1234", J.With("Trace", "1234455"))
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
				J.Info("1234")
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
				J.Info("1234", J.With("Trace", "1234455"))
			}
		})
	})
}

func TestTimeAndSizeRotation(t *testing.T) {
	// 定义我们的测试场景
	testCases := []struct {
		name     string // 子测试的名称
		compress bool   // 是否开启压缩
	}{
		{"WithCompression", true},
		{"WithoutCompression", false},
	}

	for _, tc := range testCases {
		// 使用 t.Run 为每个场景创建一个独立的子测试
		t.Run(tc.name, func(t *testing.T) {
			// 1. 为每个子测试设置独立的环境
			tempDir := t.TempDir()
			originalDefaultPath := DefaultPath
			DefaultPath = tempDir + "/"
			defer func() { DefaultPath = originalDefaultPath }()

			// 2. 手动构建 LogOptions 以精确控制配置
			// 注意：我们不使用 InitJsonTimeSizeLog，因为它可能硬编码了 Compress 的值。
			// 在测试中，手动构建依赖项是更好的实践。
			logOpts := New()
			logOpts.SetDivision(TimeAndSizeDivision)
			// 我们的 NewTimeSizeRotator 会自动处理日期和后缀，这里只提供基础文件名
			logOpts.SetInfoFile(filepath.Join(tempDir, "ts_test_ts_info"))
			logOpts.MaxSize = 1 // 1MB，用于轻松触发轮转
			logOpts.MaxBackups = 10
			logOpts.MaxAge = 28
			logOpts.Compress = tc.compress // <-- 从测试场景中获取 Compress 配置
			logOpts.TimeUnit = Day
			logOpts.CloseConsoleDisplay() // 测试时关闭控制台输出，避免干扰

			l := logOpts.InitLogger()

			// 3. 写入足够的数据（约3MB）以确保至少轮转2次
			logMessage := strings.Repeat("a", 1024) // 1KB
			for i := 0; i < 3072; i++ {             // 3MB
				l.Info(logMessage)
			}

			// 等待异步的压缩和清理操作完成
			time.Sleep(3 * time.Second)

			// 4. 动态确定预期的文件扩展名
			expectedBackupExt := ".log"
			if tc.compress {
				expectedBackupExt = ".log.gz"
			}

			// 5. 构造预期的文件列表
			dateStr := time.Now().Format("20060102")
			expectedFiles := map[string]bool{
				// 当前正在写入的日志文件永远是 .log，不会被压缩
				fmt.Sprintf("ts_test_ts_info.%s.log", dateStr): false,
				// 第一个备份文件，其扩展名根据 Compress 配置决定
				fmt.Sprintf("ts_test_ts_info.%s-1%s", dateStr, expectedBackupExt): false,
				// 第二个备份文件
				fmt.Sprintf("ts_test_ts_info.%s-2%s", dateStr, expectedBackupExt): false,
			}

			// 6. 验证文件
			t.Logf("在目录 '%s' 中查找文件 (Compress=%v):", tempDir, tc.compress)
			files, err := os.ReadDir(tempDir)
			if err != nil {
				t.Fatalf("无法读取日志目录: %v", err)
			}

			foundFiles := 0
			for _, f := range files {
				t.Logf("  找到文件: %s", f.Name())
				if _, ok := expectedFiles[f.Name()]; ok {
					expectedFiles[f.Name()] = true
					foundFiles++
				}
			}

			// 如果找到的文件数量少于预期，测试失败
			if foundFiles < len(expectedFiles) {
				for name, found := range expectedFiles {
					if !found {
						t.Errorf("期望的日志文件未找到: %s", name)
					}
				}
			} else {
				t.Log("所有预期的日志文件均已成功创建！")
			}
		})
	}
}
