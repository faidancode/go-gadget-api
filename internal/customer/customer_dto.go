package customer

import (
	"go-gadget-api/internal/pkg/response"
)

type ListCustomerRequest struct {
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	Search string `json:"search"`
}

type UpdateProfileRequest struct {
	Name            *string `json:"name"`
	Password        *string `json:"password"`
	CurrentPassword *string `json:"current_password"`
}

type CustomerDetailsRequest struct {
	CustomerID string
}

type CustomerAddressesRequest struct {
	CustomerID string
	Page       int
	Limit      int
	Search     string
}

type CustomerOrdersRequest struct {
	CustomerID string
	Page       int
	Limit      int
	Search     string
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
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
}

type UpdateStatusRequest struct {
	IsActive *bool `json:"is_active" validate:"required"`
}

type CustomerDetailResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
}

type PaginatedAddressResponse struct {
	Data []AddressResponse       `json:"data"`
	Meta response.PaginationMeta `json:"meta"`
}

type PaginatedOrderResponse struct {
	Data []OrderResponse         `json:"data"`
	Meta response.PaginationMeta `json:"meta"`
}

type AddressResponse struct {
	ID           string `json:"id"`
	AddressName  string `json:"addressName"`
	ReceiverName string `json:"receiverName"`
	Phone        string `json:"phone"`
	FullAddress  string `json:"fullAddress"`
	IsMain       bool   `json:"isMain"`
}

type OrderResponse struct {
	ID          string `json:"id"`
	OrderNumber string `json:"orderNumber"`
	TotalPrice  int    `json:"totalPrice"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
}
