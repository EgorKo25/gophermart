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
	Number     string  `json:"number,order,omitempty"`
	Status     string  `json:"status,omitempty"`
	Accrual    float64 `json:"accrual,sum,omitempty"`
	UploadedAt string  `json:"uploaded_at,omitempty"`
}
