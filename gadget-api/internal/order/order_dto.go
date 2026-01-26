package order

import "time"

// ==================== REQUEST STRUCTS ====================

type CheckoutRequest struct {
	UserID    string `json:"-"`
	AddressID string `json:"addressdId" binding:"required"`
	Note      string `json:"note"`
}

type ListOrderRequest struct {
	UserID string `json:"-"`
	Page   int32  `json:"page"`
	Limit  int32  `json:"limit"`
	Status string `json:"status"` // filter by status
}

type ListOrderAdminRequest struct {
	Page   int32  `json:"page"`
	Limit  int32  `json:"limit"`
	Status string `json:"status"`  // filter by status
	UserID string `json:"userdId"` // filter by user
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type UpdateStatusAdminRequest struct {
	Status    string  `json:"status" binding:"required"`
	ReceiptNo *string `json:"receiptdNo"`
}

// ==================== RESPONSE STRUCTS ====================

type CheckoutResponse struct {
	ID          string    `json:"id"`
	OrderNumber string    `json:"orderdNumber"`
	Status      string    `json:"status"`
	TotalPrice  float64   `json:"totalPrice"`
	PlacedAt    time.Time `json:"placedAt"`
}

type OrderResponse struct {
	ID          string              `json:"id"`
	OrderNumber string              `json:"orderdNumber"`
	Status      string              `json:"status"`
	ReceiptNo   *string             `json:"receiptdNo,omitempty"` // Tambahkan di sini
	TotalPrice  float64             `json:"totalPrice"`
	PlacedAt    time.Time           `json:"placedAt"`
	Items       []OrderItemResponse `json:"items,omitempty"`
}

type OrderItemResponse struct {
	ProductID    string  `json:"productdId"`
	NameSnapshot string  `json:"name"`
	UnitPrice    float64 `json:"unitPrice"`
	Quantity     int32   `json:"quantity"`
	Subtotal     float64 `json:"subtotal"` // unitPrice * quantity
}

type OrderDetailResponse struct {
	ID          string              `json:"id"`
	OrderNumber string              `json:"orderdNumber"`
	UserID      string              `json:"userdId"`
	Status      string              `json:"status"`
	TotalPrice  float64             `json:"totalPrice"`
	Note        string              `json:"note"`
	PlacedAt    time.Time           `json:"placedAt"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
	Items       []OrderItemResponse `json:"items"`
}

type ListOrderResponse struct {
	Orders []OrderResponse `json:"orders"`
	Total  int64           `json:"total"`
	Page   int32           `json:"page"`
	Limit  int32           `json:"limit"`
}

type ListOrderAdminResponse struct {
	Orders []OrderAdminResponse `json:"orders"`
	Total  int64                `json:"total"`
	Page   int32                `json:"page"`
	Limit  int32                `json:"limit"`
}

type OrderAdminResponse struct {
	ID          string    `json:"id"`
	OrderNumber string    `json:"orderdNumber"`
	UserID      string    `json:"userdId"`
	UserEmail   string    `json:"userEmail,omitempty"` // jika perlu join dengan user table
	Status      string    `json:"status"`
	TotalPrice  float64   `json:"totalPrice"`
	PlacedAt    time.Time `json:"placedAt"`
}
