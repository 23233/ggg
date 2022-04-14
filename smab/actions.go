package smab

import "go.mongodb.org/mongo-driver/bson/primitive"

// 自定义action

type SmAction struct {
	DefaultField `bson:",inline,flatten"`
	CreateUserId primitive.ObjectID `json:"create_user_id" bson:"create_user_id" comment:"创建者"`
	UserId       primitive.ObjectID `json:"user_id" bson:"user_id" comment:"用户"`
	Scope        string             `json:"scope" bson:"scope" comment:"作用范围"` // 表名
	Name         string             `json:"name" bson:"name" comment:"操作名称"`
	Scheme       string             `json:"scheme" bson:"scheme" comment:"表单定义" mab:"t=textarea"`
	PostUrl      string             `json:"post_url" bson:"post_url" comment:"发送接口"`
}
