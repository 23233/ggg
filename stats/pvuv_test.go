package stats

import (
	"context"
	"fmt"
	"github.com/23233/ggg/ut"
	"github.com/go-redis/redis/v8"
	"os"
	"strconv"
	"testing"
	"time"
)

var rdb *redis.Client

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func initRdb() *redis.Client {
	db, _ := strconv.Atoi(getEnv("REDISDB", "2"))
	return GenRedisClient(getEnv("REDISHOST", "127.0.0.1:6379"),
		getEnv("REDISPD", ""), db)
}

func TestMain(m *testing.M) {
	rdb = initRdb()
	err := rdb.Ping(context.TODO()).Err()
	if err != nil {
		panic(fmt.Errorf("redis连接失败:%s", err))
	}
	m.Run()
	_ = rdb.Close()
}

func TestNewStats(t *testing.T) {
	ctx := context.Background()
	m := NewStats("all", rdb)
	m.MustAdd(ctx, "今天1")
	m.MustAdd(ctx, "今天2")
	m.MustAdd(ctx, "今天3")
	m.MustAdd(ctx, "今天4")
	oKey := m.GenerateKey(time.Now().AddDate(0, 0, -1).Format("2006-01-02"))
	m.MustAddAny(ctx, oKey, "昨天1", "昨天5", "昨天9")
	m.MustAddAny(ctx, oKey, "昨天2", "昨天6", "昨天10")
	m.MustAddAny(ctx, oKey, "昨天3", "昨天7", "昨天11")
	m.MustAddAny(ctx, oKey, "昨天4", "昨天8", "昨天12")
	// 获取上个月第一天
	mt := ut.GetFirstDateOfMonth().AddDate(0, -1, 0)
	mKey := m.GenerateKey(mt.Format("2006-01-02"))
	m.MustAddAny(ctx, mKey, "上月1")
	m.MustAddAny(ctx, mKey, "上月2")
	m.MustAddAny(ctx, mKey, "上月3")
	m.MustAddAny(ctx, mKey, "上月4")
	m.MustAddAny(ctx, mKey, "上月5")

	// 测试汇总
	count, err := m.NowCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("今日总数:%d 初次测试应该是4", count)
	// 测试本星期
	mc, err := m.GetNowWeekCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("本星期总数:%d ", mc)
	// 测试上月总数
	sm, err := m.DayTimeRangerCount(ctx, mt, ut.GetFirstDateOfMonth())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("上月总数:%d  初次测试应该是5", sm)
	// 测试汇总本月
	mzb, err := m.GetAnyMonthCount(ctx, time.Now().Month())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("测试本月汇总:%d  初次测试应该是16", mzb)

}

func TestNewStatsKey(t *testing.T) {
	ctx := context.Background()
	m := NewStatsKey("article", rdb, "idlonglengthaaaa")
	m.MustAdd(ctx, "今天1", "今天6")
	m.MustAdd(ctx, "今天1", "今天7")
	m.MustAdd(ctx, "今天2", "今天8")
	m.MustAdd(ctx, "今天3", "今天9")
	m.MustAdd(ctx, "今天4", "今天10")
	m.MustAdd(ctx, "今天5", "今天11")
	// 测试汇总
	count, err := m.NowCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("总数:%d", count)

}

func BenchmarkNewStats(b *testing.B) {
	ctx := context.Background()
	m := NewStats("article", rdb)

	for i := 0; i < b.N; i++ {
		m.MustAdd(ctx, "随机111")
	}
}
