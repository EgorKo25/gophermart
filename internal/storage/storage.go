package storage

type User struct {
	Login  string `json:"login"`
	Passwd string `json:"password"`
}

type Order struct {
	ID         int     `json:"omitempty"`
	User       string  `json:"omitempty"`
	Number     string  `json:"number,omitempty"`
	Status     string  `json:"status,omitempty"`
	Accrual    float64 `json:"accrual,omitempty"`
	UploadedAt string  `json:"uploaded_at"`
}

func NewUser(login string, passwd string) *User {

	return &User{
		Login:  login,
		Passwd: passwd,
	}
}
