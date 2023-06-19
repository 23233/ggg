package pipe

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/redis/rueidis"
	"math"
	"strconv"
	"strings"
	"time"
)

type SwipeItem struct {
	Id string    `json:"board"`
	C  int8      `json:"c"` // 随机参数数量
	N  bool      `json:"n"` // true 参数在请求包中传递+号  false 参数在header中传递
	P  string    `json:"p"` // 前缀
	T  time.Time `json:"t"` // 获取时间
	X  int       `json:"x"` // 滑块中心位置
	Y  int       `json:"y"` // 滑块中心位置
	B  int       `json:"b"` // 滑块大小
}

func (c *SwipeItem) ToString() string {
	var st = make([]string, 0, 10)
	st = append(st, c.Id)
	cFlag := strconv.Itoa(int(c.C))
	if c.N {
		cFlag = "+" + cFlag
	} else {
		cFlag = "-" + cFlag
	}
	st = append(st, cFlag)
	st = append(st, c.P)
	st = append(st, strconv.Itoa(c.X))
	st = append(st, strconv.Itoa(c.Y))
	st = append(st, strconv.Itoa(c.B))
	return strings.Join(st, ",")
}

func (c *SwipeItem) ParseStr(raw string) error {
	rawList := strings.Split(raw, ",")
	if len(rawList) != 6 {
		return errors.New("参数错误")
	}
	c.Id = rawList[0]
	cFlag := rawList[1]
	nStr := cFlag[:1]
	cCount, _ := strconv.Atoi(strings.TrimPrefix(cFlag, nStr))
	c.C = int8(cCount)
	c.N = nStr == "+"
	c.P = rawList[2]
	c.X, _ = strconv.Atoi(rawList[3])
	c.Y, _ = strconv.Atoi(rawList[4])
	c.B, _ = strconv.Atoi(rawList[5])
	return nil
}

type SwipeValid struct {
	Sid          string    `json:"sid,omitempty"`
	RefreshCount int       `json:"refresh_count,omitempty"` // 刷新次数
	X            int64     `json:"x,omitempty"`             // 停止时的x 滑块左侧上角
	Y            int64     `json:"y,omitempty"`             // 停止时的y 滑块左侧上角
	T            time.Time `json:"t,omitempty"`             // 滑动开始的时间
	Te           time.Time `json:"te,omitempty"`            // 滑动结束的时间
	S            float64   `json:"s,omitempty"`             // 拖动速度
	N            float64   `json:"n,omitempty"`             // 归一化 拖动X平均值波动值最高
	Nm           float64   `json:"nm,omitempty"`            // 归一化 拖动X平均值波动值最低
}

// SwipeValidCode 滑块验证码实例
type SwipeValidCode struct {
	RandomCount   int8      `json:"random_count,omitempty"` //
	Prefix        string    `json:"Prefix,omitempty"`       // 前缀
	EnableTime    time.Time `json:"enable_time,omitempty"`  // 生效时间
	ExpireSec     int64     `json:"expire_sec,omitempty"`   // 过期秒数
	drawMaxWidth  int       // 暂定300
	drawMaxHeight int       // 暂定160
	blockSize     int       // 滑块大小

	// 历史记录
	history []SwipeValidCode

	// db连接
	rdb rueidis.Client
}

func NewSwipeValidInst(rdb rueidis.Client) *SwipeValidCode {
	var m = new(SwipeValidCode)
	m.RandomCount = 3
	m.Prefix = "S"
	m.EnableTime = time.Now()
	m.ExpireSec = int64((3 * time.Minute).Seconds())
	m.drawMaxWidth = 300
	m.drawMaxHeight = 160
	m.blockSize = 45
	m.rdb = rdb
	return m
}

func (c *SwipeValidCode) BlockSize(half bool) int {
	if half {
		return c.blockSize / 2
	}
	return c.blockSize
}

func (c *SwipeValidCode) RandomBlockX() int {
	return ut.RandomInt(c.BlockSize(false)*2, c.drawMaxWidth-(c.BlockSize(false)*2))
}

func (c *SwipeValidCode) RandomBlockY() int {
	return ut.RandomInt(c.BlockSize(true), c.drawMaxHeight-c.BlockSize(false))
}

// Gen 生成滑块要素并存入redis
func (c *SwipeValidCode) Gen(ctx context.Context) (*SwipeItem, error) {
	var d = new(SwipeItem)
	d.Id = GenUUid()
	d.C = c.RandomCount
	d.P = c.Prefix
	d.N = ut.RandomInt(1, 2) > 1
	d.T = time.Now()
	d.X = c.RandomBlockX()
	d.Y = c.RandomBlockY()
	d.B = c.blockSize
	marshal, err := json.Marshal(&d)
	if err != nil {
		return nil, err
	}

	// board 写入redis
	resp := c.rdb.Do(ctx, c.rdb.B().Set().Key("sv:"+d.Id).Value(string(marshal)).ExSeconds(c.ExpireSec).Build())
	if resp.Error() != nil {
		return nil, err
	}

	return d, nil
}

func (c *SwipeValidCode) Check(ctx iris.Context, raw string) (*SwipeValid, error) {
	if len(raw) < 1 {
		return nil, errors.New("参数无法获取")
	}
	// 对数据包进行base64
	decodeStr, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, errors.Wrap(err, "参数结构验证失败")
	}
	// 解构
	var item = new(SwipeValid)
	err = json.Unmarshal(decodeStr, item)
	if err != nil {
		return nil, errors.Wrap(err, "参数结构解析失败")
	}
	// 进行验证 首先判断id是否存在
	resp := c.rdb.Do(ctx, c.rdb.B().Get().Key("sv:"+item.Sid).Build())
	if resp.Error() != nil {
		return nil, err
	}
	dataPack, err := resp.ToString()
	if err != nil {
		return nil, err
	}
	var dataItem SwipeItem
	err = json.Unmarshal([]byte(dataPack), &dataItem)
	if err != nil {
		return nil, err
	}

	// 进行验证

	// 开始和结束的时间必须有一定的间隔 ms
	if item.T.Sub(item.Te).Milliseconds() > 300 {
		return nil, errors.New("拖动过快")
	}

	// 随机参数验证
	count := 0
	if dataItem.N {
		// 验证请求包中参数 只看原始请求包
		rawBody := make(map[string]any)
		err = json.Unmarshal(decodeStr, rawBody)
		if err != nil {
			return nil, errors.Wrap(err, "原始请求参数体解构失败")
		}
		for k := range rawBody {
			if strings.HasPrefix(k, dataItem.P+"-") {
				count += 1
				continue
			}
		}
	} else {
		// 验证header中的参数
		for k := range ctx.Request().Header {
			if strings.HasPrefix(k, dataItem.P+"-") {
				count += 1
				continue
			}
		}
	}

	if count != int(dataItem.C) {
		return nil, errors.New("安全策略验证失败")
	}

	// 刷新次数验证
	if item.RefreshCount >= 5 {
		return nil, errors.New("刷新次数超出")
	}

	// 获取时间超过了5分钟
	if time.Now().After(dataItem.T.Add(5 * time.Minute)) {
		return nil, errors.New("时间过期")
	}

	// 停止位置的 X 距离差
	leftX := float64(dataItem.X) - float64(dataItem.B/2)
	if math.Abs(float64(item.X)-leftX) > 10 {
		return nil, errors.New("位置未命中")
	}

	// 对于验证正确的 直接删除key 不用关心删除结果
	_ = c.rdb.Do(ctx, c.rdb.B().Del().Key("sv:"+item.Sid).Build())

	return item, nil

}

func (c *SwipeValidCode) SetRandomCount(newCount int8) {
	c.history = append(c.history, *c)
	c.RandomCount = newCount
	c.EnableTime = time.Now()
}

func (c *SwipeValidCode) SetPrefix(prefix string) {
	c.history = append(c.history, *c)
	c.Prefix = prefix
	c.EnableTime = time.Now()
}

func (c *SwipeValidCode) SetExpireSec(newSec int64) {
	c.history = append(c.history, *c)
	c.ExpireSec = newSec
	c.EnableTime = time.Now()
}
