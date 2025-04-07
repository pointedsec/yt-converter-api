package models

type User struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Role          string `json:"role"` // 'admin' o 'guest'
	Active        bool   `json:"active"`
	Created_at    string `json:"created_at"`
	Updated_at    string `json:"updated_at"`
	Last_login_at string `json:"last_login_at"`
}
