package smab

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SmDashBoardScreen 仪表台屏幕 只有管理员能够操作
type SmDashBoardScreen struct {
	DefaultField `bson:",inline,flatten"`
	Priority     uint64               `json:"priority" bson:"priority"`     // 优先级 越大越提前
	Name         string               `json:"name" bson:"name"`             // 屏幕名称
	IsDefault    bool                 `json:"is_default" bson:"is_default"` // 是否为默认
	CreateUserId primitive.ObjectID   `json:"create_user_id" bson:"create_user_id"`
	ViewUserId   []primitive.ObjectID `json:"view_user_id" bson:"view_user_id"` // 有权查看的用户
}

type dashBoardExtra struct {
	Width  uint64 `json:"width" bson:"width"`   // 宽度
	Height uint64 `json:"height" bson:"height"` // 高度
	X      uint64 `json:"x" bson:"x"`           // x位置
	Y      uint64 `json:"y" bson:"y"`           // y位置
}

// SmDashBoard 图表
type SmDashBoard struct {
	DefaultField  `bson:",inline,flatten"`
	ScreenId      primitive.ObjectID `json:"screen_id" bson:"screen_id"` // 屏幕ID
	Name          string             `json:"name" bson:"name"`           // 图表名称
	ChatType      string             `json:"chat_type" bson:"chat_type"` // 图表类型
	DataUri       string             `json:"data_uri" bson:"data_uri"`   // 数据请求接口
	Extra         dashBoardExtra     `json:"extra" bson:"extra"`
	Config        string             `json:"config" bson:"config"`                 // 配置文件 json字符串
	RefreshSecond uint64             `json:"refresh_second" bson:"refresh_second"` // 数据刷新间隔 0则是不刷新
	CreateUserId  primitive.ObjectID `json:"create_user_id" bson:"create_user_id"`
}
