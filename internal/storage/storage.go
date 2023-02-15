package storage

type User struct {
	Firstname string
	Lastname  string
	Login     string
	Passwd    string
	Orders    []Order
	Balance   float64
}

type Order struct {
	Title     string
	UserToken string
	Balls     float64
}

func NewUser(firstname string, lastname string, login string, passwd string) *User {

	return &User{
		Firstname: firstname,
		Lastname:  lastname,
		Login:     login,
		Passwd:    passwd,
	}
}
