package contentSafe

import "testing"

func TestWxImgV1(t *testing.T) {

	pass, err := wxImgCheckV1("https://cdn.golangdocs.com/wp-content/uploads/2020/09/Download-Files-for-Golang.png")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("req测试通过 %v", pass)

}

func TestAutoHitImg(t *testing.T) {
	pass, err := AutoHitImg("https://cdn.golangdocs.com/wp-content/uploads/2020/09/Download-Files-for-Golang.png")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("图片测试通过吗? %v", pass)
}
