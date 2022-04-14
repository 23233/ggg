package ut

import (
	"testing"
	"time"
)

func TestRetryFunc(t *testing.T) {
	err := RetryFunc(func() error {
		return nil
	}, 10, 10*time.Second)
	if err != nil {
		t.Error(err)
	}
}
