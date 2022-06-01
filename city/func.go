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

// GetZcsStr 获取直辖市
func GetZcsStr(hasSuffix bool) []string {
	if hasSuffix {
		return []string{"北京市", "上海市", "重庆市", "天津市"}
	}
	return []string{"北京", "上海", "重庆", "天津"}
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

func init() {
	rand.Seed(time.Now().UTC().UnixNano()) // always seed random!
}
