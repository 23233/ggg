package smab

type taskChangeSuccessReq struct {
	Id      string `json:"id" validate:"required"`
	Success bool   `json:"success" bson:"success"`
}
