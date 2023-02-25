package storage

type User struct {
	Login    string  `json:"login"`
	Passwd   string  `json:"password"`
	Balance  float64 `json:"current"`
	Withdraw float64 `json:"withdraw"`
}

type Order struct {
	ID         int
	User       string  `json:"user,omitempty"`
	Number     string  `json:"number"`
	Status     string  `json:"status"`
	Accrual    float64 `json:"accrual"`
	UploadedAt string  `json:"uploaded_at"`
}
