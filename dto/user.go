package dto

type User struct {
	ID               string `json:"id"`
	Username         string `json:"username"`
	Avatar           string `json:"avatar"`
	Bot              bool   `json:"bot"`
	IsYou            bool   `json:"is_you"`
	Status           int    `json:"status"`
	UnionOpenID      string `json:"union_openid"`
	UnionUserAccount string `json:"union_user_account"`
	MemberOpenID     string `json:"member_openid"`
	UserOpenID       string `json:"user_openid"`
	MemberRole       string `json:"member_role"`
	Scope            string `json:"scope"`
}
