package logger

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// TimeSizeRotator 实现了 io.Writer 接口，
// 能够同时根据日期和文件大小来轮转日志。
// 当天内的轮转会以 "-1", "-2" 的序号形式命名。
type TimeSizeRotator struct {
	mu         sync.Mutex
	filename   string // 日志文件的基础名，例如 "ts_test_ts_info"
	maxSize    int    // 单个文件的最大大小 (MB)
	maxBackups int    // 最多保留的备份数量
	maxAge     int    // 最长保留天数
	compress   bool   // 是否压缩
	timeUnit   TimeUnit

	currentFile *os.File
	currentSize int64
	currentDate string // 当前文件的日期戳，格式 "20060102"
	sequence    int    // 当天的文件序号
}

// NewTimeSizeRotator 是 TimeSizeRotator 的构造函数
func NewTimeSizeRotator(filename string, maxSize, maxBackups, maxAge int, compress bool, unit TimeUnit) io.Writer {
	r := &TimeSizeRotator{
		filename:   filename,
		maxSize:    maxSize,
		maxBackups: maxBackups,
		maxAge:     maxAge,
		compress:   compress,
		timeUnit:   unit,
	}
	// 初始化时主动进行一次轮转，以创建第一个日志文件
	r.rotate()
	return r
}

// Write 实现了 io.Writer 接口
func (r *TimeSizeRotator) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查日期是否变化
	now := time.Now()
	dateStr := now.Format("20060102")
	if r.currentDate != dateStr {
		r.rotate()
	}

	// 检查大小是否超限
	// r.maxSize 的单位是 MB，所以要乘以 1024 * 1024
	if r.currentFile != nil && r.currentSize+int64(len(p)) > int64(r.maxSize)*1024*1024 {
		r.rotate()
	}

	if r.currentFile == nil {
		return 0, fmt.Errorf("log file not open")
	}

	n, err = r.currentFile.Write(p)
	if err != nil {
		return n, err
	}
	r.currentSize += int64(n)

	return n, nil
}

// rotate 执行轮转操作
// 这个方法必须在锁的保护下调用
func (r *TimeSizeRotator) rotate() error {
	// 1. 如果有旧文件，关闭并重命名
	if r.currentFile != nil {
		err := r.currentFile.Close()
		if err != nil {
			// 即使关闭失败，也继续尝试轮转
			fmt.Fprintf(os.Stderr, "failed to close current log file: %v\n", err)
		}

		// 构造备份文件名，例如 ts_test_ts_info.20250813-1.log
		backupFilename := fmt.Sprintf("%s.%s-%d.log", r.filename, r.currentDate, r.sequence)
		err = os.Rename(r.currentFile.Name(), backupFilename)
		if err != nil {
			return fmt.Errorf("failed to rename log file: %v", err)
		}

		// 如果需要压缩，则异步执行压缩
		if r.compress {
			go r.compressFile(backupFilename)
		}
	}

	// 2. 更新日期和序号
	now := time.Now()
	dateStr := now.Format("20060102")
	if r.currentDate != dateStr {
		r.currentDate = dateStr
		r.sequence = 1 // 新的一天，序号重置为1
	} else {
		r.sequence++ // 当天内的下个文件，序号递增
	}

	// 3. 打开新的日志文件
	// 新文件的名字不带序号，例如 ts_test_ts_info.20250813.log
	newFilename := fmt.Sprintf("%s.%s.log", r.filename, r.currentDate)
	file, err := os.OpenFile(newFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open new log file: %v", err)
	}
	r.currentFile = file

	// 获取新文件大小（如果是已存在的文件）
	stat, err := file.Stat()
	if err == nil {
		r.currentSize = stat.Size()
	} else {
		r.currentSize = 0
	}

	// 4. 清理旧的备份文件
	go r.cleanup()

	return nil
}

// compressFile 压缩指定的日志文件
func (r *TimeSizeRotator) compressFile(filename string) {
	in, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file for compression: %v\n", err)
		return
	}
	defer in.Close()

	out, err := os.Create(filename + ".gz")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create gzipped log file: %v\n", err)
		return
	}
	defer out.Close()

	gz := gzip.NewWriter(out)
	defer gz.Close()

	if _, err := io.Copy(gz, in); err != nil {
		fmt.Fprintf(os.Stderr, "failed to gzip log file: %v\n", err)
		// 如果压缩失败，保留原始文件，删除空的.gz文件
		os.Remove(out.Name())
		return
	}

	// 压缩成功后删除原始文件
	os.Remove(filename)
}

// cleanup 清理旧的备份文件
func (r *TimeSizeRotator) cleanup() {
	dir := filepath.Dir(r.filename)
	base := filepath.Base(r.filename)

	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var backups []string

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		// 匹配格式如 "basename.YYYYMMDD-N.log" 或 "basename.YYYYMMDD-N.log.gz"
		if strings.HasPrefix(f.Name(), base+".") && (strings.HasSuffix(f.Name(), ".log") || strings.HasSuffix(f.Name(), ".log.gz")) {
			// 从文件名中提取日期
			parts := strings.Split(f.Name(), ".")
			if len(parts) > 1 {
				dateStr := parts[len(parts)-2]
				if strings.Contains(dateStr, "-") {
					dateStr = strings.Split(dateStr, "-")[0]
				}

				fileTime, err := time.Parse("20060102", dateStr)
				if err != nil {
					continue
				}

				// 检查文件是否过期 (maxAge)
				if r.maxAge > 0 && time.Since(fileTime) > time.Duration(r.maxAge)*24*time.Hour {
					os.Remove(filepath.Join(dir, f.Name()))
					continue
				}
				backups = append(backups, f.Name())
			}
		}
	}

	// 检查备份数量是否超限 (maxBackups)
	if r.maxBackups > 0 && len(backups) > r.maxBackups {
		// 按文件名排序，旧文件在前
		sort.Strings(backups)

		filesToRemove := backups[:len(backups)-r.maxBackups]
		for _, f := range filesToRemove {
			os.Remove(filepath.Join(dir, f))
		}
	}
}
