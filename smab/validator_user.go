package smab

type permissions struct {
	Scope  string `json:"scope"`
	Action string `json:"action"`
}

// UserLoginReq 用户登录
type UserLoginReq struct {
	UserName string `json:"user_name" comment:"用户名" validate:"required,max=20,min=3"`
	Password string `json:"password" comment:"密码" validate:"required,min=3,max=20"`
}

// UserChangePasswordReq 用户变更密码
type UserChangePasswordReq struct {
	Id       string `json:"id" comment:"id" validate:"required"`
	Password string `json:"password" comment:"密码" validate:"required,min=6,max=20"`
}

// 新增用户
type addUserReq struct {
	Name        string               `json:"name" validate:"required,max=60"`
	Password    string               `json:"password" validate:"required,min=6,max=100"`
	Desc        string               `json:"desc"`
	Phone       string               `json:"phone"`
	SuperUser   bool                 `json:"super_user"`
	Permissions []permissions        `json:"permissions"`
	QianKun     []QianKunConfigExtra `json:"qian_kun" validate:"omitempty"`
	FilterData  []FilterDataExtra    `json:"filter_data" validate:"omitempty"`
}

type editUserReq struct {
	Id         string               `json:"id" comment:"id" validate:"required"`
	Desc       string               `json:"desc"`
	Phone      string               `json:"phone"`
	SuperUser  bool                 `json:"super_user"`
	QianKun    []QianKunConfigExtra `json:"qian_kun" validate:"omitempty"`
	FilterData []FilterDataExtra    `json:"filter_data" validate:"omitempty"`
}

type editUserPermissionsReq struct {
	Id          string        `json:"id" comment:"id" validate:"required"`
	Permissions []permissions `json:"permissions"`
}

// admin 变更用户群组
type UserChangeRolesReq struct {
	Id   uint64 `json:"id" comment:"id" validate:"required"`
	Role string `json:"role" comment:"群组名" validate:"required"`
	Add  bool   `json:"add" comment:"添加"`
}
