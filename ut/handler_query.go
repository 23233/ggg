package ut

import (
	"fmt"
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

const OpRegex = "regex"
const DefaultUidTag = "uid"

var AllowOps = []string{"eq", "gt", "gte", "lt", "lte", "ne", "in", "nin", OpRegex}

var (
	NotEnableGlobalSearch = errors.New("未启用全局搜索")
)

type Kov struct {
	Key   string `json:"key,omitempty" bson:"key,omitempty"` // 格式是 "abc.die.ede"
	Op    string `json:"op,omitempty" bson:"op,omitempty"`   // 格式为英文 看 AllowOps
	Value any    `json:"value,omitempty" bson:"value,omitempty"`
}

type GeoItem struct {
	Field   string  `json:"field,omitempty"`    // 获取的字段
	ToField string  `json:"to_field,omitempty"` // 距离字段名
	Lng     float64 `json:"lng,omitempty"`      // 范围 -180 到 180
	Lat     float64 `json:"lat,omitempty"`      // 范围 -90 到 90
	GeoMax  int64   `json:"geo_max,omitempty"`  // 单位为米
	GeoMin  int64   `json:"geo_min,omitempty"`  // 单位为米
}

type Pk struct {
	LocalKey      string `json:"local_key,omitempty"`
	RemoteModelId string `json:"remote_model_id,omitempty"`
	RemoteKey     string `json:"remote_key,omitempty"`
	Alias         string `json:"alias,omitempty"`
	EmptyReturn   bool   `json:"empty_return,omitempty"`
	Unwind        bool   `json:"unwind,omitempty"`
}

type QueryParse struct {
	And  []*Kov   `json:"and,omitempty"`
	Or   []*Kov   `json:"or,omitempty"`
	Geos *GeoItem `json:"geos,omitempty"`
}

func (c *QueryParse) InsertOrReplaces(target string, data ...*Kov) {

	for _, k := range data {
		switch target {
		case "and":
			c.And = c.insertOrReplace(c.And, k)
		default:
			c.Or = c.insertOrReplace(c.Or, k)
		}
	}

}

func (c *QueryParse) insertOrReplace(dataList []*Kov, now *Kov) []*Kov {
	has := false
	result := dataList
	if result == nil {
		result = make([]*Kov, 0)
	}

	for _, k := range dataList {
		if k.Key == now.Key {
			has = true
			k.Value = now.Value
			break
		}
	}
	if !has {
		result = append(result, now)
	}
	return result
}

type BasePage struct {
	Page     int64 `json:"page,omitempty"`
	PageSize int64 `json:"page_size,omitempty"`
}

type BaseSort struct {
	SortDesc []string `json:"sort_desc,omitempty"`
	SortAsc  []string `json:"sort_asc,omitempty"`
}

type BaseQuery struct {
	*BasePage
	*BaseSort
}

type QueryFull struct {
	*QueryParse
	*BaseQuery
	Pks      []*Pk `json:"pks,omitempty"`
	GetCount bool  `json:"get_count,omitempty"` // 获取过滤总数
}

type PruneCtxQuery struct {
	params          map[string]string
	lastKey         string
	searchKey       string
	allowOps        []string
	opSuffix        string
	inlineFieldsSep string
	orPrefix        string
	geoKey          string
	allEmbedKeys    []string
	geoDistanceKey  string

	minLength int
	maxLength int
}

// PruneParseUrlParams 纯解析url上的参数
func (p *PruneCtxQuery) PruneParseUrlParams() (and []*Kov, or []*Kov, err error) {
	if p.maxLength < 1 {
		p.init()
	}
	for k, v := range p.params {
		bk := k

		has := false
		for _, key := range p.allEmbedKeys {
			if bk == key {
				has = true
				break
			}
		}
		if has {
			continue
		}

		// 如果有or的前缀 则去掉
		if strings.HasPrefix(k, p.orPrefix) {
			bk = strings.TrimPrefix(bk, p.orPrefix)
		}
		// 判断操作符_是否存在
		opIndex := strings.LastIndex(bk, p.opSuffix)
		var op = ""
		// 如果操作符存在去找到对应的操作
		var lenK = len(k)
		if opIndex >= lenK-p.maxLength && opIndex <= lenK-p.minLength {
			for _, allowOp := range p.allowOps {
				// 组合成 _op
				suffix := p.opSuffix + allowOp
				if strings.HasSuffix(bk, suffix) {
					bk = strings.TrimSuffix(bk, suffix)
					op = allowOp
					break
				}
			}
		}

		var value any = v
		if op == "in" || op == "nin" {
			value = strings.Split(v, ",")
		}

		var kop = new(Kov)
		kop.Op = op
		kop.Value = value
		// url过滤完成后转换为程序正确识别的.
		kop.Key = strings.ReplaceAll(bk, p.inlineFieldsSep, ".")

		if strings.HasPrefix(k, p.orPrefix) {
			or = append(or, kop)
			continue
		}
		and = append(and, kop)
		delete(p.params, k)
	}
	return
}

// PruneParseQuery 解析出 _last geo search
func (p *PruneCtxQuery) PruneParseQuery(searchFields []string, geoKey string) (*QueryParse, error) {
	mqp := new(QueryParse)

	// 解析 _last
	lastUid, ok := p.params[p.lastKey]
	if ok {
		lastSort, ok := p.params["_lastSort"]
		if !ok {
			lastSort = "gt"
		}

		mqp.And = append(mqp.And, &Kov{
			Key:   DefaultUidTag,
			Op:    lastSort,
			Value: lastUid,
		})
	}

	// 解析geo
	if len(geoKey) >= 1 {
		geo, ok := p.params[p.geoKey]
		if ok {
			if !strings.Contains(geo, ",") {
				return nil, fmt.Errorf("%s 参数格式错误", p.geoKey)
			}
			geoList := strings.Split(geo, ",")
			if len(geoList) != 2 {
				return nil, fmt.Errorf("%s 参数格式解析错误", p.geoKey)
			}
			lng, err := strconv.ParseFloat(geoList[0], 64)
			if err != nil {
				return nil, err
			}
			lat, err := strconv.ParseFloat(geoList[1], 64)
			if err != nil {
				return nil, err
			}
			var r = &GeoItem{
				Lng: lng,
				Lat: lat,
			}

			geoMax, ok := p.params["_gmax"]
			if ok {
				maxInt, err := TypeChange(geoMax, "int64")
				if err != nil {
					return nil, err
				}
				r.GeoMax = maxInt.(int64)
			}

			geoMin, ok := p.params["_gmin"]
			if ok {
				minInt, err := TypeChange(geoMin, "int64")
				if err != nil {
					return nil, err
				}
				r.GeoMin = minInt.(int64)
			}
			r.Field = geoKey
			r.ToField = p.geoDistanceKey

			mqp.Geos = r

		}
	}

	// 解析搜索
	search, ok := p.params[p.searchKey]
	if ok {
		if searchFields == nil || len(searchFields) < 0 {
			// 这里是否抛出错误有待商榷
			return nil, NotEnableGlobalSearch
		}
		// _s __在左右则为模糊 _s=__赵日天
		// 仅支持字符串
		searchVal := search
		v := strings.ReplaceAll(searchVal, p.inlineFieldsSep, "")

		for _, field := range searchFields {
			pattern := v

			// 如果是前匹配
			if strings.HasPrefix(searchVal, p.inlineFieldsSep) {
				pattern = "^" + v
			} else if strings.HasSuffix(searchVal, p.inlineFieldsSep) {
				// 如果是后匹配
				pattern = v + "$"
			}

			var kop = new(Kov)
			kop.Key = field
			kop.Op = OpRegex
			kop.Value = pattern
			mqp.Or = append(mqp.Or, kop)
		}

	}

	return mqp, nil
}

// PruneParsePage 解析出 page page_size sort
func (p *PruneCtxQuery) PruneParsePage() (*BaseQuery, error) {
	q := new(BaseQuery)
	q.BasePage = new(BasePage)
	q.BaseSort = new(BaseSort)

	page, ok := p.params["page"]
	if ok {
		pageInt, err := TypeChange(page, "int64")
		if err == nil {
			q.Page = pageInt.(int64)
		} else {
			q.Page = 1
		}
	}

	pageSize, ok := p.params["page_size"]
	if ok {
		pageSizeInt, err := TypeChange(pageSize, "int64")
		if err == nil {
			q.PageSize = pageSizeInt.(int64)
		} else {
			q.PageSize = 10
		}
	}

	descStr, ok := p.params["_od"]
	if ok {
		descField := strings.Split(descStr, ",")
		q.SortDesc = descField
	}

	ascStr, ok := p.params["_o"]
	if ok {
		orderBy := strings.Split(ascStr, ",")
		q.SortAsc = orderBy
	}

	return q, nil

}

func (p *PruneCtxQuery) PruneParse(params map[string]string, searchFields []string, geoKey string) (*QueryFull, error) {
	p.params = params

	// 解析出query and 和 or
	query, err := p.PruneParseQuery(searchFields, geoKey)
	if err != nil {
		return nil, err
	}

	// 解析字段
	and, or, err := p.PruneParseUrlParams()
	if err != nil {
		return nil, err
	}

	query.And = append(query.And, and...)
	query.Or = append(query.Or, or...)

	// 解析出 page page_size
	base, err := p.PruneParsePage()
	if err != nil {
		return nil, err
	}
	mapper := new(QueryFull)
	mapper.BaseQuery = base
	mapper.QueryParse = query

	return mapper, nil

}

func (p *PruneCtxQuery) SetParams(params map[string]string) {
	p.params = params
}

func (p *PruneCtxQuery) init() {
	// 初始化最小值和最大值为第一个字符串的长度
	minLength := len(p.allowOps[0])
	maxLength := len(p.allowOps[0])
	// 遍历切片，更新最小值和最大值
	for _, op := range p.allowOps {
		opLength := len(op)
		if opLength < minLength {
			minLength = opLength
		}
		if opLength > maxLength {
			maxLength = opLength
		}
	}
	// 加符号的位置
	p.minLength = minLength + 1
	p.maxLength = maxLength + 1
}

func NewPruneCtxQuery() *PruneCtxQuery {
	m := &PruneCtxQuery{
		lastKey:         "_last",
		searchKey:       "_s",
		allowOps:        AllowOps,
		opSuffix:        "_",
		inlineFieldsSep: "__",
		orPrefix:        "_o_",
		geoKey:          "_g",
		geoDistanceKey:  "_distance",
		allEmbedKeys:    []string{"_last", "_s", "_g", "_od", "_o", "_gmin", "_gmax", "_lastSort", "page", "page_size"},
	}
	m.init()
	return m
}
