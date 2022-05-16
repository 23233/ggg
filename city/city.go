package city

import (
	"math/rand"
	"strings"
	"time"
)

var (
	// 直辖市
	cq = []string{
		"重庆",
		"万州区",
		"涪陵区",
		"渝中区",
		"大渡口区",
		"江北区",
		"沙坪坝区",
		"九龙坡区",
		"南岸区",
		"北碚区",
		"綦江区",
		"大足区",
		"渝北区",
		"巴南区",
		"黔江区",
		"长寿区",
		"江津区",
		"合川区",
		"永川区",
		"南川区",
		"璧山区",
		"铜梁区",
		"潼南区",
		"荣昌区",
		"开州区",
		"梁平区",
		"武隆区",
		"城口县",
		"丰都县",
		"垫江县",
		"忠县",
		"云阳县",
		"奉节县",
		"巫山县",
		"巫溪县",
		"石柱县", //石柱土家族自治县
		"秀山县", //秀山土家族苗族自治县
		"酉阳县", //酉阳土家族苗族自治县
		"彭水县", //彭水苗族土家族自治县
	}
	bj = []string{
		"北京",
		"东城区",
		"西城区",
		"朝阳区",
		"丰台区",
		"石景山区",
		"海淀区",
		"门头沟区",
		"房山区",
		"通州区",
		"顺义区",
		"昌平区",
		"大兴区",
		"怀柔区",
		"平谷区",
		"密云区",
		"延庆区",
	}
	sh = []string{
		"上海",
		"黄浦区",
		"徐汇区",
		"长宁区",
		"静安区",
		"普陀区",
		"虹口区",
		"杨浦区",
		"闵行区",
		"宝山区",
		"嘉定区",
		"浦东新区",
		"金山区",
		"松江区",
		"青浦区",
		"奉贤区",
	}
	tj = []string{
		"天津",
		"和平区",
		"河东区",
		"河西区",
		"南开区",
		"河北区",
		"红桥区",
		"东丽区",
		"西青区",
		"津南区",
		"北辰区",
		"武清区",
		"宝坻区",
		"滨海新区",
		"宁河区",
		"静海区",
		"蓟州区",
	}
	// 港澳台
	gat = []string{
		"香港",
		"澳门",
		"台湾",
	}
	// 河北省
	hbs = []string{
		"河北",
		"石家庄市",
		"唐山市",
		"秦皇岛市",
		"邯郸市",
		"邢台市",
		"保定市",
		"张家口市",
		"承德市",
		"沧州市",
		"廊坊市",
		"衡水市",
	}
	// 山西省
	sxs = []string{
		"山西",
		"太原市",
		"大同市",
		"阳泉市",
		"长治市",
		"晋城市",
		"朔州市",
		"晋中市",
		"运城市",
		"忻州市",
		"临汾市",
		"吕梁市",
	}
	// 内蒙古 仅保留内蒙即可
	nmg = []string{
		"内蒙古",
		"呼和浩特市",
		"包头市",
		"乌海市",
		"赤峰市",
		"通辽市",
		"鄂尔多斯市",
		"呼伦贝尔市",
		"巴彦淖尔市",
		"乌兰察布市",
		"兴安盟",
		"锡林郭勒盟",
		"阿拉善盟",
	}
	// 辽宁省
	lns = []string{
		"辽宁",
		"沈阳市",
		"大连市",
		"鞍山市",
		"抚顺市",
		"本溪市",
		"丹东市",
		"锦州市",
		"营口市",
		"阜新市",
		"辽阳市",
		"盘锦市",
		"铁岭市",
		"朝阳市",
		"葫芦岛市",
		"长春市",
		"吉林市",
		"四平市",
		"辽源市",
		"通化市",
		"白山市",
		"松原市",
		"白城市",
		"延边朝鲜族自治州",
	}
	// 吉林
	jls = []string{
		"吉林",
		"长春市",
		"吉林市",
		"四平市",
		"辽源市",
		"通化市",
		"白山市",
		"松原市",
		"白城市",
		"延边朝鲜族自治州",
	}
	// 黑龙江
	hlj = []string{
		"黑龙江",
		"哈尔滨市",
		"齐齐哈尔市",
		"鸡西市",
		"鹤岗市",
		"双鸭山市",
		"大庆市",
		"伊春市",
		"佳木斯市",
		"七台河市",
		"牡丹江市",
		"黑河市",
		"绥化市",
		"大兴安岭地区",
	}
	// 江苏
	js = []string{
		"江苏",
		"南京市",
		"无锡市",
		"徐州市",
		"常州市",
		"苏州市",
		"南通市",
		"连云港市",
		"淮安市",
		"盐城市",
		"扬州市",
		"镇江市",
		"泰州市",
		"宿迁市",
	}
	// 浙江
	zjs = []string{
		"浙江",
		"杭州市",
		"宁波市",
		"温州市",
		"嘉兴市",
		"湖州市",
		"绍兴市",
		"金华市",
		"衢州市",
		"舟山市",
		"台州市",
		"丽水市",
	}
	// 安徽
	ahs = []string{
		"安徽",
		"合肥市",
		"芜湖市",
		"蚌埠市",
		"淮南市",
		"马鞍山市",
		"淮北市",
		"铜陵市",
		"安庆市",
		"黄山市",
		"滁州市",
		"阜阳市",
		"宿州市",
		"六安市",
		"亳州市",
		"池州市",
		"宣城市",
	}
	// 福建
	fjs = []string{
		"福建",
		"福州市",
		"厦门市",
		"莆田市",
		"三明市",
		"泉州市",
		"漳州市",
		"南平市",
		"龙岩市",
		"宁德市",
	}
	// 江西
	jxs = []string{
		"江西",
		"南昌市",
		"景德镇市",
		"萍乡市",
		"九江市",
		"新余市",
		"鹰潭市",
		"赣州市",
		"吉安市",
		"宜春市",
		"抚州市",
		"上饶市",
	}
	// 山东
	sdx = []string{
		"山东",
		"济南市",
		"青岛市",
		"淄博市",
		"枣庄市",
		"东营市",
		"烟台市",
		"潍坊市",
		"济宁市",
		"泰安市",
		"威海市",
		"日照市",
		"临沂市",
		"德州市",
		"聊城市",
		"滨州市",
		"菏泽市",
	}
	// 河南
	hns = []string{
		"河南",
		"郑州市",
		"开封市",
		"洛阳市",
		"平顶山市",
		"安阳市",
		"鹤壁市",
		"新乡市",
		"焦作市",
		"濮阳市",
		"许昌市",
		"漯河市",
		"三门峡市",
		"南阳市",
		"商丘市",
		"信阳市",
		"周口市",
		"驻马店市",
	}
	// 湖北
	hubs = []string{
		"湖北",
		"武汉市",
		"黄石市",
		"十堰市",
		"宜昌市",
		"襄阳市",
		"鄂州市",
		"荆门市",
		"孝感市",
		"荆州市",
		"黄冈市",
		"咸宁市",
		"随州市",
		"恩施土家族苗族自治州",
	}
	// 湖南
	hunans = []string{
		"湖南",
		"长沙市",
		"株洲市",
		"湘潭市",
		"衡阳市",
		"邵阳市",
		"岳阳市",
		"常德市",
		"张家界市",
		"益阳市",
		"郴州市",
		"永州市",
		"怀化市",
		"娄底市",
		"湘西土家族苗族自治州",
	}
	// 广东
	gdx = []string{
		"广东",
		"广州市",
		"韶关市",
		"深圳市",
		"珠海市",
		"汕头市",
		"佛山市",
		"江门市",
		"湛江市",
		"茂名市",
		"肇庆市",
		"惠州市",
		"梅州市",
		"汕尾市",
		"河源市",
		"阳江市",
		"清远市",
		"东莞市",
		"中山市",
		"潮州市",
		"揭阳市",
		"云浮市",
	}
	// 广西
	gxs = []string{
		"广西",
		"南宁市",
		"柳州市",
		"桂林市",
		"梧州市",
		"北海市",
		"防城港市",
		"钦州市",
		"贵港市",
		"玉林市",
		"百色市",
		"贺州市",
		"河池市",
		"来宾市",
		"崇左市",
	}
	// 海南
	hains = []string{
		"海南",
		"海口市",
		"三亚市",
		"三沙市",
		"儋州市",
	}
	// 四川
	scx = []string{
		"四川",
		"成都市",
		"自贡市",
		"攀枝花市",
		"泸州市",
		"德阳市",
		"绵阳市",
		"广元市",
		"遂宁市",
		"内江市",
		"乐山市",
		"南充市",
		"眉山市",
		"宜宾市",
		"广安市",
		"达州市",
		"雅安市",
		"巴中市",
		"资阳市",
		"阿坝藏族羌族自治州",
		"甘孜藏族自治州",
		"凉山彝族自治州",
	}
	// 贵州
	gzx = []string{
		"贵州",
		"贵阳市",
		"六盘水市",
		"遵义市",
		"安顺市",
		"毕节市",
		"铜仁市",
		"黔西南布依族苗族自治州",
		"黔东南苗族侗族自治州",
		"黔南布依族苗族自治州",
	}
	// 云南
	yns = []string{
		"云南",
		"昆明市",
		"曲靖市",
		"玉溪市",
		"保山市",
		"昭通市",
		"丽江市",
		"普洱市",
		"临沧市",
		"楚雄彝族自治州",
		"红河哈尼族彝族自治州",
		"文山壮族苗族自治州",
		"西双版纳傣族自治州",
		"大理白族自治州",
		"德宏傣族景颇族自治州",
		"怒江傈僳族自治州",
		"迪庆藏族自治州",
	}
	// 西藏
	xzs = []string{
		"西藏",
		"拉萨市",
		"日喀则市",
		"昌都市",
		"林芝市",
		"山南市",
		"那曲市",
		"阿里地区",
	}
	// 陕西
	shananx = []string{
		"陕西",
		"西安市",
		"铜川市",
		"宝鸡市",
		"咸阳市",
		"渭南市",
		"延安市",
		"汉中市",
		"榆林市",
		"安康市",
		"商洛市",
	}
	// 甘肃
	gans = []string{
		"甘肃",
		"兰州市",
		"嘉峪关市",
		"金昌市",
		"白银市",
		"天水市",
		"武威市",
		"张掖市",
		"平凉市",
		"酒泉市",
		"庆阳市",
		"定西市",
		"陇南市",
		"临夏回族自治州", // 临夏回族自治州
		"甘南藏族自治州", // 甘南藏族自治州
	}
	// 青海
	qhs = []string{
		"青海",
		"西宁市",
		"海东市",
		"海北藏族自治州",    // 海北藏族自治州
		"黄南藏族自治州",    //黄南藏族自治州
		"海南藏族自治州",    // 海南藏族自治州
		"果洛藏族自治州",    // 果洛藏族自治州
		"玉树藏族自治州",    // 玉树藏族自治州
		"海西蒙古族藏族自治州", //海西蒙古族藏族自治州
	}
	// 宁夏
	nxs = []string{
		"宁夏",
		"银川市",
		"石嘴山市",
		"吴忠市",
		"固原市",
		"中卫市",
	}
	// 新疆
	xjs = []string{
		"新疆",
		"乌鲁木齐市",
		"克拉玛依市",
		"吐鲁番市",
		"哈密市",
		"昌吉回族自治州",   // 昌吉回族自治州
		"博尔塔拉蒙古自治州", // 博尔塔拉蒙古自治州
		"巴音郭楞蒙古自治州", // 巴音郭楞蒙古自治州
		"阿克苏地区",
		"克孜勒苏柯尔克孜自治州", // 克孜勒苏柯尔克孜自治州
		"喀什地区",
		"和田地区",
		"伊犁哈萨克自治州", // 伊犁哈萨克自治州
		"塔城地区",
		"阿勒泰地区",
	}
)

var (
	provinceKey = "province" // 省份的key
	gatKey      = "gat"      // 港澳台的key
	zxsKey      = "zxs"      // 直辖市的key
)

var cityList []TreeNode

type TreeNode struct {
	Name     string   `json:"name"`
	Group    string   `json:"group"`
	Children []string `json:"children"`
}

// RandomGet 随机取n条 exclude包含即排除
func RandomGet(count uint8, exclude ...string) []string {
	var r = make([]string, 0, count)
	for i := 0; i < int(count); i++ {
		one := randomGetOne(exclude...)
		r = append(r, one)
	}
	return r
}

func randomGetOne(exclude ...string) string {
	var one string
	for true {
		index := rand.Intn(len(cityList))
		c := cityList[index]
		one = c.Children[rand.Intn(len(c.Children))]
		has := false
		for _, s := range exclude {
			if strings.Contains(one, s) {
				has = true
				break
			}
		}
		if !has {
			return one
		}
	}
	return one
}

// GetZcs 获取直辖市
func GetZcs() []string {
	return []string{"北京", "上海", "广州", "天津"}
}

// GetGyt 获取港澳台
func GetGyt() []string {
	return []string{"香港", "澳门", "台湾"}
}

// GetProvince 获取省份
func GetProvince(hasSuffix bool) []string {
	var l []string
	for _, node := range cityList {
		if node.Group == provinceKey {
			if hasSuffix {
				l = append(l, node.Name)
			} else {
				l = append(l, strings.TrimSuffix(node.Name, "省"))
			}
		}
	}
	return l
}

// GetAllCity 获取所有市
func GetAllCity(hasZcs bool) []string {
	var r []string
	for _, node := range cityList {
		if node.Group == provinceKey {
			r = append(r, node.Children...)
		} else if hasZcs && node.Group == "zxs" {
			r = append(r, node.Children...)
		}
	}
	return r
}

// GetAll 获取所有
func GetAll() []TreeNode {
	return cityList
}

// SetNewTree 使用新的数据替换
func SetNewTree(l []TreeNode) {
	cityList = l
}

// ProvinceGetCity 省份名称获取下属城市
func ProvinceGetCity(ProvinceName string) ([]string, bool) {
	for _, node := range cityList {
		if node.Group == provinceKey {
			return node.Children, true
		}
	}
	return nil, false
}

func newProvince(name string, children []string) TreeNode {
	return TreeNode{
		Name:     name,
		Group:    provinceKey,
		Children: children,
	}
}

func InitLoadCityData() []TreeNode {
	// 4大直辖市下属 重庆 北京 上海 天津
	// 香港 澳门 台湾
	// 河北 山西 内蒙古 辽宁 吉林 黑龙江 江苏 浙江 安徽 福建 江西
	// 山东 河南 湖北 湖南 广东 广西 海南 四川 贵州 云南 西藏 陕西 甘肃 青海 宁夏 新疆
	var treeNodes []TreeNode
	treeNodes = append(treeNodes, TreeNode{
		Name:     "重庆",
		Group:    zxsKey,
		Children: cq[1:],
	})
	treeNodes = append(treeNodes, TreeNode{
		Name:     "北京",
		Group:    zxsKey,
		Children: bj[1:],
	})
	treeNodes = append(treeNodes, TreeNode{
		Name:     "上海",
		Group:    zxsKey,
		Children: sh[1:],
	})
	treeNodes = append(treeNodes, TreeNode{
		Name:     "天津",
		Group:    zxsKey,
		Children: tj[1:],
	})
	treeNodes = append(treeNodes, TreeNode{
		Name:     "港澳台",
		Group:    gatKey,
		Children: gat,
	})
	treeNodes = append(treeNodes, newProvince("河北省", hbs[1:]))
	treeNodes = append(treeNodes, newProvince("山西省", sxs[1:]))
	treeNodes = append(treeNodes, newProvince("内蒙古省", nmg[1:]))
	treeNodes = append(treeNodes, newProvince("辽宁省", lns[1:]))
	treeNodes = append(treeNodes, newProvince("吉林省", jls[1:]))
	treeNodes = append(treeNodes, newProvince("黑龙江省", hlj[1:]))
	treeNodes = append(treeNodes, newProvince("江苏省", js[1:]))
	treeNodes = append(treeNodes, newProvince("浙江省", zjs[1:]))
	treeNodes = append(treeNodes, newProvince("安徽省", ahs[1:]))
	treeNodes = append(treeNodes, newProvince("福建省", fjs[1:]))
	treeNodes = append(treeNodes, newProvince("江西省", jxs[1:]))
	treeNodes = append(treeNodes, newProvince("山东省", sdx[1:]))
	treeNodes = append(treeNodes, newProvince("河北省", hbs[1:]))
	treeNodes = append(treeNodes, newProvince("河南省", hns[1:]))
	treeNodes = append(treeNodes, newProvince("湖北省", hubs[1:]))
	treeNodes = append(treeNodes, newProvince("湖南省", hunans[1:]))
	treeNodes = append(treeNodes, newProvince("广东省", gdx[1:]))
	treeNodes = append(treeNodes, newProvince("广西省", gxs[1:]))
	treeNodes = append(treeNodes, newProvince("海南省", hains[1:]))
	treeNodes = append(treeNodes, newProvince("四川省", scx[1:]))
	treeNodes = append(treeNodes, newProvince("贵州省", gzx[1:]))
	treeNodes = append(treeNodes, newProvince("云南省", yns[1:]))
	treeNodes = append(treeNodes, newProvince("西藏省", xzs[1:]))
	treeNodes = append(treeNodes, newProvince("陕西省", shananx[1:]))
	treeNodes = append(treeNodes, newProvince("甘肃省", gans[1:]))
	treeNodes = append(treeNodes, newProvince("青海省", qhs[1:]))
	treeNodes = append(treeNodes, newProvince("宁夏省", nxs[1:]))
	treeNodes = append(treeNodes, newProvince("新疆省", xjs[1:]))
	return treeNodes
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano()) // always seed random!
	cityList = InitLoadCityData()
}
