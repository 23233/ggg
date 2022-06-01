package city

import (
	"math/rand"
	"strings"
	"time"
)

// RandomGet 随机取n条 exclude包含即排除
func RandomGet(count uint8, exclude ...string) []TreeNode {
	var r = make([]TreeNode, 0, count)
	for i := 0; i < int(count); i++ {
		one := randomGetOne(exclude...)
		if one != nil {
			r = append(r, *one)
		}
	}
	return r
}

func randomGetOne(exclude ...string) *TreeNode {
	for true {
		index := rand.Intn(len(FlatData))
		c := FlatData[index]
		has := false
		for _, s := range exclude {
			if strings.Contains(c.Name, s) {
				has = true
				break
			}
		}
		if !has {
			return c
		}
	}
	return nil
}

// GetCodes 合并获取多个code内容
func GetCodes(codes ...string) []TreeNode {
	if len(codes) >= 1 {
		var r = make([]TreeNode, 0)
		for _, code := range codes {
			result := AdCodeGet(code)
			if result != nil {
				r = append(r, *result)
			}
		}
		return r
	}
	return nil
}

func GetZsc() []TreeNode {
	return GetCodes("110100", "310000", "500100", "120100")
}

// GetZcsStr 获取直辖市
func GetZcsStr(hasSuffix bool) []string {
	result := GetZsc()
	var r = make([]string, 0, 4)
	for _, node := range result {
		if hasSuffix {
			r = append(r, node.Name+node.Suffix)
		} else {
			r = append(r, node.Name)
		}
	}
	return r
}

func GetZscBase() []TreeBase {
	result := GetZsc()
	var r = make([]TreeBase, 0, len(result))
	for _, node := range result {
		r = append(r, node.TreeBase)
	}
	return r
}

// GetGytStr 获取港澳台
func GetGytStr() []string {
	return []string{"香港", "澳门", "台湾"}
}

// GetProvince 获取所有省份
func GetProvince() []TreeNode {
	var p = make([]TreeNode, 0, len(TreeData))
	for _, province := range TreeData {
		var item TreeNode
		item = *province
		item.Children = nil
		p = append(p, item)
	}
	return p
}

// ProvinceGetCity 省份名称获取下属城市
func ProvinceGetCity(provinceName string) ([]TreeNode, bool) {
	for _, province := range TreeData {
		var fullName = province.Name + province.Suffix
		if province.Name == provinceName || fullName == provinceName {
			var items = make([]TreeNode, 0, len(province.Children))

			for _, child := range province.Children {
				var item TreeNode
				item = *child
				item.Children = nil
				items = append(items, item)
			}

			return items, true
		}
	}
	return nil, false
}

func ProvinceGetCityStr(provinceName string, hasSuffix bool) ([]string, bool) {

	citys, has := ProvinceGetCity(provinceName)
	if has {
		var r = make([]string, 0, len(citys))

		for _, node := range citys {
			if hasSuffix {
				r = append(r, node.Name+node.Suffix)
			} else {
				r = append(r, node.Name)
			}
		}

		return r, has
	}
	return nil, false

}

// GetAllCity 获取所有市
func GetAllCity() []TreeNode {
	var data = make([]TreeNode, 0, len(FlatData))
	for _, node := range FlatData {
		if node.IsCity {
			data = append(data, *node)
		}
	}
	return data
}

func GetAllCityStr(hasSuffix bool) []string {
	var allCity = GetAllCity()
	var r = make([]string, 0, len(allCity))
	for _, node := range allCity {
		if hasSuffix {
			r = append(r, node.Name+node.Suffix)
		} else {
			r = append(r, node.Name)
		}
	}
	return r
}

func GetAllCityBase() []TreeBase {
	var allCity = GetAllCity()
	var r = make([]TreeBase, 0, len(allCity))
	for _, node := range allCity {
		r = append(r, node.TreeBase)
	}
	return r
}

func AdCodeGet(code string) *TreeNode {
	var r TreeNode
	for _, node := range FlatData {
		if node.Adcode == code {
			r = *node
			return &r
		}
	}
	return nil
}

func Search(name string) []TreeNode {
	var r = make([]TreeNode, 0)
	for _, node := range FlatData {
		var fullName = node.Name + node.Suffix
		if node.Name == name || fullName == name {
			r = append(r, *node)
		}
	}
	return r
}

func Match(name string) (*TreeNode, bool) {
	var r TreeNode
	for _, node := range FlatData {
		var fullName = node.Name + node.Suffix
		if node.Name == name || fullName == name {
			r = *node
			return &r, true
		}
	}
	return nil, false
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano()) // always seed random!
}
