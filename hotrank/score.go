package hotrank

import "time"

// 评分标准
var workScore = map[string]int64{
	"like":           100, // 点赞
	"comment":        50,  // 评论/回复
	"share":          100, // 分享
	"view_user":      10,  // 用户查看
	"view_anonymous": 1,   // 匿名用户查看
}

// 评分的验证规则 若未注册验证事件则默认不通过 则不加分
var scoreTrigger = map[string]func(*WorkCall) bool{
	"like": func(c *WorkCall) bool {
		return !c.hotHistoryExists()
	},
	"view_user": func(c *WorkCall) bool {
		_, err := workGc.Get(c.FromUserId)
		return err != nil
	},
	"view_anonymous": func(c *WorkCall) bool {
		_, err := workGc.Get(c.ip)
		return err != nil
	},
	"comment": func(c *WorkCall) bool {
		return c.FromUserId != c.ToUserId
	},
	"share": func(c *WorkCall) bool {
		return c.FromUserId != c.ToUserId
	},
}

// 评分新增后操作
var scoreAddAfter = map[string]func(*WorkCall){
	"view_user": func(c *WorkCall) {
		_ = workGc.SetWithExpire(c.FromUserId, "1", time.Minute)
	},
	"view_anonymous": func(c *WorkCall) {
		_ = workGc.SetWithExpire(c.ip, "1", time.Minute)
	},
}

//AddScore 新增评分标准
func AddScore(k string, v int64) {
	workScore[k] = v
}

// AddScoreValid 新增事件验证
func AddScoreValid(k string, fc func(call *WorkCall) bool) {
	scoreTrigger[k] = fc
}

// AddScoreAfter 新增评分增加成功过后回调
func AddScoreAfter(k string, fc func(call *WorkCall)) {
	scoreAddAfter[k] = fc
}
