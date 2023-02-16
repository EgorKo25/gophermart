package storage

type User struct {
	Login  string `json:"login"`
	Passwd string `json:"password"`
}

type Order struct {
	Title     string
	UserToken string
	Balls     float64
}

func NewUser(login string, passwd string) *User {

	return &User{
		Login:  login,
		Passwd: passwd,
	}
}
