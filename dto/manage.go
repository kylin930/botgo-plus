package dto

type GroupManageData struct {
	Timestamp      uint64 `json:"timestamp,omitempty"`
	GroupOpenID    string `json:"group_openid,omitempty"`
	OpMemberOpenID string `json:"op_member_openid,omitempty"`
}

type C2CManageData struct {
	Timestamp uint64 `json:"timestamp,omitempty"`
	OpenID    string `json:"openid,omitempty"`
}
