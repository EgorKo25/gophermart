package storage

type User struct {
	Login  string `json:"login"`
	Passwd string `json:"password"`
}

type Order struct {
	User    string  `json:"omitempty"`
	Number  int     `json:"order,omitempty"`
	Status  string  `json:"status,omitempty"`
	Accrual float64 `json:"accrual,omitempty"`
}

func NewUser(login string, passwd string) *User {

	return &User{
		Login:  login,
		Passwd: passwd,
	}
}
