package customer

type UpdateProfileRequest struct {
	Name            *string `json:"name"`
	Password        *string `json:"password"`
	CurrentPassword *string `json:"current_password"`
}

type CustomerResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CustomerListResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
}

type UpdateStatusRequest struct {
	IsActive bool `json:"is_active" validate:"required"`
}

type CustomerDetailResponse struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Email     string            `json:"email"`
	Phone     string            `json:"phone"`
	IsActive  bool              `json:"is_active"`
	CreatedAt string            `json:"created_at"`
	Addresses []AddressResponse `json:"addresses,omitempty"`
	Orders    []OrderResponse   `json:"orders,omitempty"`
}

type AddressResponse struct {
	ID           string `json:"id"`
	AddressName  string `json:"address_name"`
	ReceiverName string `json:"receiver_name"`
	Phone        string `json:"phone"`
	FullAddress  string `json:"full_address"`
	IsMain       bool   `json:"is_main"`
}

type OrderResponse struct {
	ID          string `json:"id"`
	OrderNumber string `json:"order_number"`
	TotalAmount int    `json:"total_amount"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}
