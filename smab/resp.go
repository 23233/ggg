package smab

type permissionsRespItem struct {
	Title      string                `json:"title"`
	Key        string                `json:"key"`
	V          string                `json:"-"`
	Alias      string                `json:"alias"`
	Group      string                `json:"group,omitempty"`
	GroupLevel uint                  `json:"group_level,omitempty"`
	Level      uint                  `json:"level,omitempty"`
	Children   []permissionsRespItem `json:"children,omitempty"`
}

type permissionsResp struct {
	Data []permissionsRespItem `json:"data"`
}
