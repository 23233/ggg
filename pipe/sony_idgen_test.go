package pipe

import "testing"

func TestSfNextId(t *testing.T) {
	sfId := SfNextId()
	println(sfId)
	sfId = SfNextId()
	println(sfId)
	sfId = SfNextId()
	println(sfId)
}
