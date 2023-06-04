package ua

import "testing"

func TestGet(t *testing.T) {
	result := Get()
	t.Logf(result)
}

func TestGetMobile(t *testing.T) {
	r := GetMobile()
	t.Logf(r)
}

func BenchmarkGet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Get()
	}
}
