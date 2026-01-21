package order

import "time"

// ==================== REQUEST STRUCTS ====================

type CheckoutRequest struct {
	UserID    string `json:"-"`
	AddressID string `json:"address_id" binding:"required"`
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
	UserID string `json:"user_id"` // filter by user
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type UpdateStatusAdminRequest struct {
	Status    string  `json:"status" binding:"required"`
	ReceiptNo *string `json:"receipt_no"`
}

// ==================== RESPONSE STRUCTS ====================

type CheckoutResponse struct {
	ID          string    `json:"id"`
	OrderNumber string    `json:"order_number"`
	Status      string    `json:"status"`
	TotalPrice  float64   `json:"total_price"`
	PlacedAt    time.Time `json:"placed_at"`
}

type OrderResponse struct {
	ID          string              `json:"id"`
	OrderNumber string              `json:"order_number"`
	Status      string              `json:"status"`
	ReceiptNo   *string             `json:"receipt_no,omitempty"` // Tambahkan di sini
	TotalPrice  float64             `json:"total_price"`
	PlacedAt    time.Time           `json:"placed_at"`
	Items       []OrderItemResponse `json:"items,omitempty"`
}

type OrderItemResponse struct {
	ProductID    string  `json:"product_id"`
	NameSnapshot string  `json:"name"`
	UnitPrice    float64 `json:"unit_price"`
	Quantity     int32   `json:"quantity"`
	Subtotal     float64 `json:"subtotal"` // unit_price * quantity
}

type OrderDetailResponse struct {
	ID          string              `json:"id"`
	OrderNumber string              `json:"order_number"`
	UserID      string              `json:"user_id"`
	Status      string              `json:"status"`
	TotalPrice  float64             `json:"total_price"`
	Note        string              `json:"note"`
	PlacedAt    time.Time           `json:"placed_at"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
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
	OrderNumber string    `json:"order_number"`
	UserID      string    `json:"user_id"`
	UserEmail   string    `json:"user_email,omitempty"` // jika perlu join dengan user table
	Status      string    `json:"status"`
	TotalPrice  float64   `json:"total_price"`
	PlacedAt    time.Time `json:"placed_at"`
}
