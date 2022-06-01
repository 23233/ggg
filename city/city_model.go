package city

import (
	_ "embed"
	"fmt"
	"github.com/jszwec/csvutil"
	"github.com/mozillazg/go-pinyin"
	"strings"
)

//go:embed only_city.csv
var cityCsv []byte

var py = pinyin.NewArgs()

type TreeBase struct {
	Adcode string `json:"adcode"` // 唯一id 数字的
	Name   string `json:"name"`   // 名称
	Suffix string `json:"suffix"` // 后缀
}

type TreeNode struct {
	TreeBase
	Parent     string      `json:"parent"`      // 这个是父级的adcode
	Py         string      `json:"py"`          // 拼音
	Pf         string      `json:"pf"`          // 拼音首字符
	IsProvince bool        `json:"is_province"` // 省份
	IsCity     bool        `json:"is_city"`     // 城市
	IsNyc      bool        `json:"is_nyc"`      // 直辖市
	Lat        float64     `json:"lat" `
	Lng        float64     `json:"lng"`
	Children   []*TreeNode `json:"children"`
}

type csvMap struct {
	Adcode string  `json:"adcode" csv:"adcode"`
	Name   string  `json:"name" csv:"name"`
	Suffix string  `json:"suffix" csv:"suffix"`
	Lat    float64 `json:"lat" csv:"lat"`
	Lng    float64 `json:"lng" csv:"lng"`
	Parent string  `json:"parent" csv:"parent"`
	Level  string  `json:"level" csv:"level"` // province city district
}

var record = make([]csvMap, 0)

var TreeData = make([]*TreeNode, 0)

var FlatData = make([]*TreeNode, 0)

func loadCsv() error {
	if len(record) > 1 {
		return nil
	}
	if err := csvutil.Unmarshal(cityCsv, &record); err != nil {
		fmt.Println("error:", err)
		return err
	}

	provinceList := make([]*TreeNode, 0)

	// 获取所有省份
	for _, c := range record {
		if c.Level == "province" {
			provinceList = append(provinceList, mapToTree(c))
		}
	}

	// 根据省份获取所有城市
	for _, province := range provinceList {
		for _, c := range record {
			if c.Parent == province.Adcode {
				province.Children = append(province.Children, mapToTree(c))
			}
		}
	}

	// 根据省份下的城市获取所有街道
	for _, province := range provinceList {
		for _, city := range province.Children {
			for _, c := range record {
				if c.Parent == city.Adcode {
					city.Children = append(city.Children, mapToTree(c))
				}
			}
		}
	}

	TreeData = provinceList

	for _, c := range record {
		FlatData = append(FlatData, mapToTree(c))
	}

	return nil
}

func isNyc(name string) bool {
	if name == "北京" || name == "上海" || name == "重庆" || name == "天津" {
		return true
	}
	return false
}

func mapToTree(c csvMap) *TreeNode {
	var st strings.Builder
	var pf strings.Builder
	for _, s := range pinyin.Pinyin(c.Name+c.Suffix, py) {
		st.WriteString(strings.Join(s, ""))
		pf.WriteString(s[0][:1])
	}
	var zxs = isNyc(c.Name)

	var province = false
	if c.Level == "province" {
		province = true
	}

	var city = false
	if c.Level == "city" {
		if !zxs {
			city = true
		}
	}

	if c.Level == "district" {
		city = true
	}

	var r = &TreeNode{
		Parent:     c.Parent,
		Py:         st.String(),
		Pf:         pf.String(),
		IsProvince: province,
		IsCity:     city,
		IsNyc:      zxs,
		Lat:        c.Lat,
		Lng:        c.Lng,
		Children:   []*TreeNode{},
	}

	r.Adcode = c.Adcode
	r.Name = c.Name
	r.Suffix = c.Suffix

	return r
}

func init() {
	_ = loadCsv()
}
