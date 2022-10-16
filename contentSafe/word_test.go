package contentSafe

import "testing"

func TestWxTextCheckV1(t *testing.T) {
	_, err, msg := wxTextCheckV1("特3456书 yuuo 莞6543李 zxcz 蒜7782法 fgnv 级\n完2347全 dfji 试3726测 asad 感3847知 qwez 到")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(msg)
}

func TestAutoHitText(t *testing.T) {
	ok, _ := AutoHitText("性爱djifw图库")
	if ok {
		t.Fatal("接口出错")
	}
}
