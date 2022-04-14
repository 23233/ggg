package mab

import "testing"

func TestUTCTrans(t *testing.T) {
	b, err := UTCTrans("2021-07-01T14:36:46.106+08:00")
	if err != nil {
		t.Fatal(err)
		return
	}
	t.Log(b)
}
