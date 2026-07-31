package dto

type User struct {
	ID               string `json:"id"`
	Username         string `json:"username"`
	Avatar           string `json:"avatar"`
	Bot              bool   `json:"bot"`
	Status           int    `json:"status"`
	UnionOpenID      string `json:"union_openid"`
	UnionUserAccount string `json:"union_user_account"`
	MemberOpenID     string `json:"member_openid"`
	UserOpenID       string `json:"user_openid"`
}
