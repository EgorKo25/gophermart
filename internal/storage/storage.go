package storage

type User struct {
	Login  string `json:"login"`
	Passwd string `json:"password"`
}

type Order struct {
	Number int
	User   string `json:"omitempty"`
}

func NewUser(login string, passwd string) *User {

	return &User{
		Login:  login,
		Passwd: passwd,
	}
}
