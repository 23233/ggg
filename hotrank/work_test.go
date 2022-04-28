package hotrank

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"
)

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

// RandomStr 随机N位字符串(英文)
func RandomStr(n int) string {
	randBytes := make([]byte, n/2)
	rand.Read(randBytes)
	return fmt.Sprintf("%x", randBytes)
}

func initRdb() *redis.Client {
	db, _ := strconv.Atoi(getEnv("REDISDB", "2"))
	return redis.NewClient(&redis.Options{
		Addr:     getEnv("REDISHOST", "127.0.0.1:6379"),
		Password: getEnv("REDISPD", ""),
		DB:       db,
		PoolSize: 100,
	})
}

func TestWorkCall(t *testing.T) {
	rdb := initRdb()
	ctx := context.Background()
	if rdb.Ping(ctx).Err() != nil {
		t.Fatal("连接redis失败")
	}
	t.Run("随机新增数据", TestAddData)
	t.Run("获取今日rank", TestGetTodayRank)
	t.Run("获取总排名", TestGetAllRank)
	t.Run("获取随机内容", TestGetRandomData)
	t.Run("获取作品排名", TestGetWorkRank)
}

func TestGetTodayRank(t *testing.T) {
	rdb := initRdb()
	instance := NewWorkHot(rdb, "")
	ctx := context.Background()
	result, err := instance.GetTodayTopRank(ctx, 20)
	if err != nil {
		t.Log("获取今日rank失败")
		t.Fatal(err)
		return
	}
	t.Logf("获取今日rank %v", result)
}

func TestGetAllRank(t *testing.T) {
	rdb := initRdb()
	instance := NewWorkHot(rdb, "")
	ctx := context.Background()
	result, err := instance.GetAllTopRank(ctx, 20)
	if err != nil {
		t.Log("获取总排行榜失败")
		t.Fatal(err)
		return
	}
	t.Logf("获取总排行榜 %v", result)
}

func TestGetRandomData(t *testing.T) {
	rdb := initRdb()
	ctx := context.Background()
	instance := NewWorkHot(rdb, "")

	r, err := instance.GetAllRandomData(ctx, 20)
	if err != nil {
		if err != EmptyError {
			t.Log("获取总随机内容失败")
			t.Fatal(err)
			return
		}
	}
	t.Logf("获取总随机内容 %v", r)

	r, err = instance.GetCityRandomData(ctx, 20, "cq")
	if err != nil {
		if err != EmptyError {
			t.Log("获取城市随机内容失败")
			t.Fatal(err)
			return
		}

	}
	t.Logf("获取城市随机内容 %v", r)

	r, err = instance.GetTimeRandomData(ctx, 20, time.Now())
	if err != nil {
		t.Log("获取今天随机内容失败")
		t.Fatal(err)
		return
	}
	t.Logf("获取今天随机内容 %v", r)

}

func TestAddData(t *testing.T) {
	rdb := initRdb()
	ctx := context.Background()

	reasons := []string{"cq", "bj"}
	for i := 0; i < 200; i++ {
		instance := NewWorkHot(rdb, "text"+RandomStr(8))
		var event = make([]string, 0, len(workScore))
		for k := range workScore {
			event = append(event, k)
		}
		for range event {
			instance.AddHotChange(ctx, event[rand.Intn(len(event))], RandomStr(12), RandomStr(8), reasons[rand.Intn(len(reasons))], "129.1.2.3")
		}

	}
}

func TestGetWorkRank(t *testing.T) {
	rdb := initRdb()
	instance := NewWorkHot(rdb, "textaa7bcd6f")
	ctx := context.Background()

	instance.AddHotChange(ctx, "comment", "jdsifijwe", "jifweiwiw", "", "")

	today, city, all := instance.GetAggregationRank(ctx)

	t.Logf("获取作品排名 今日:%f 城市:%f 所有:%f", today, city, all)

}

func init() {
	rand.Seed(time.Now().UnixNano())
}
