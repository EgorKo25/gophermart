package storage

type User struct {
	Login  string `json:"login"`
	Passwd string `json:"password"`
}

type Order struct {
	ID          int
	User        string  `json:"omitempty"`
	Number      string  `json:"order,omitempty"`
	Status      string  `json:"status,omitempty"`
	Accrual     float64 `json:"accrual,omitempty"`
	Uploaded_at string  `json:"uploaded_At"`
}

func NewUser(login string, passwd string) *User {

	return &User{
		Login:  login,
		Passwd: passwd,
	}
}
