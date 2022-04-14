package ut

import "testing"

func TestB62Str(t *testing.T) {
	s := StrToB62("6482be1b909847b78ca109fbba73e16d610ffba14dcd106d0794d08b60f508f2f7da11fc4b060bfd")
	t.Logf(s)
}

func TestHexTo58(t *testing.T) {
	s := StrToB58("6482be1b909847b78ca109fbba73e16d610ffba14dcd106d0794d08b60f508f2f7da11fc4b060bfd")
	t.Log(s)
}
