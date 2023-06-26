package ut

import (
	"errors"
	"time"
)

var MaxRetrySmallError = errors.New("重试次数需要大于1")

// RetryFunc 最大重试次数执行方法 函数返回error也会继续重试
func RetryFunc(f func() error, maxRetryCount uint, interval time.Duration) error {
	if maxRetryCount < 1 {
		return MaxRetrySmallError
	}
	var (
		nowCount uint = 1
	)
	for {
		err := f()
		if err == nil {
			return nil
		}
		if nowCount > maxRetryCount {
			return err
		}
		time.Sleep(interval)
		nowCount += 1
	}
}
