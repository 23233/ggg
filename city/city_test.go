package city

import "testing"

func TestRandomGet(t *testing.T) {
	l := RandomGet(1, "自治州")
	allCity := GetAllCityStr(true)
	t.Log(len(allCity))
	//t.Log(allCity)
	t.Log(l[0])
}

func TestGetZcsStr(t *testing.T) {
	t.Log(GetZcsStr(true))
	t.Log(GetZscBase())
}

func TestGetAllCityBase(t *testing.T) {
	allCity := GetAllCityBase()
	t.Log(len(allCity))
}

func TestAdCodeGet(t *testing.T) {
	l := AdCodeGet("130700")
	if l != nil {
		t.Log(l.Name)
	}
}

func TestProvinceGetCity(t *testing.T) {
	r, has := ProvinceGetCityStr("河北省", true)
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

func BenchmarkAdCodeGet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		AdCodeGet("130700")
	}
}

func BenchmarkGetAllCityStr(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetAllCityStr(true)
	}
}
