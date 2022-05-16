package city

import "testing"

func TestRandomGet(t *testing.T) {
	l := RandomGet(1, "自治州")
	t.Log(GetAllCity(false))
	t.Log(l[0])
}

func TestProvinceGetCity(t *testing.T) {
	r, has := ProvinceGetCity("河北省")
	if has {
		t.Log(r)
		return
	}
	t.Fatal("获取省份失败")
}

func BenchmarkRandomGet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RandomGet(10, "自治州", "地区")
	}
}
