package midtrans

type CreateTransactionRequest struct {
	OrderID     string           `json:"orderId"`
	GrossAmount int64            `json:"grossAmount"`
	Customer    *CustomerDetails `json:"customer"`
	Items       []ItemDetail     `json:"items"`
}

type CustomerDetails struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

type ItemDetail struct {
	ID    string `json:"id"`
	Price int64  `json:"price"`
	Qty   int32  `json:"qty"`
	Name  string `json:"name"`
}

type CreateTransactionResponse struct {
	Token       string `json:"snapToken"`
	RedirectURL string `json:"redirectUrl"`
}
