package ut

import "testing"

func TestRandomStr(t *testing.T) {
	t.Log(RandomStr(12))
}

func TestRandomInt(t *testing.T) {
	t.Log(RandomInt(10, 100))
}
