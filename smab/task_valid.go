package smab

type TaskInjectBaseReq struct {
	SmTaskId     string `json:"sm_task_id" form:"sm_task_id" validate:"required"`
	SmUserId     string `json:"sm_user_id" form:"sm_user_id" validate:"required"`
	SmActionName string `json:"sm_action_name" bson:"sm_action_name" validate:"required"`
}
