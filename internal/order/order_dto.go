package order

import "time"

// ==================== REQUEST STRUCTS ====================

type CheckoutRequest struct {
	AddressID string `json:"addressId" binding:"required"`
	Note      string `json:"note"`
}

type ListOrderRequest struct {
	UserID string `json:"userId"`
	Page   int32  `json:"page"`
	Limit  int32  `json:"limit"`
	Status string `json:"status"` // filter by status
}

type ListOrderAdminRequest struct {
	Page   int32  `json:"page"`
	Limit  int32  `json:"limit"`
	Status string `json:"status"` // filter by status
	UserID string `json:"userId"` // filter by user
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type UpdateStatusAdminRequest struct {
	Status    string  `json:"status" binding:"required"`
	ReceiptNo *string `json:"receiptNo"`
}

type UpdatePaymentStatusRequest struct {
	PaymentStatus string     `json:"paymentStatus" binding:"required"`
	PaymentMethod string     `json:"paymentMethod"`
	PaidAt        *time.Time `json:"paidAt"`
	CancelledAt   *time.Time `json:"cancelledAt"`
	Note          *string    `json:"note"`
}

type MidtransNotificationRequest struct {
	OrderID           string `json:"order_id" binding:"required"`
	StatusCode        string `json:"status_code" binding:"required"`
	GrossAmount       string `json:"gross_amount" binding:"required"`
	SignatureKey      string `json:"signature_key" binding:"required"`
	TransactionStatus string `json:"transaction_status" binding:"required"`
	TransactionTime   string `json:"transaction_time"`
	PaymentType       string `json:"payment_type"`
	FraudStatus       string `json:"fraud_status"`
}

type UpdatePaymentStatusInput struct {
	PaymentStatus string
	PaymentMethod string
	PaidAt        *time.Time
	CancelledAt   *time.Time
	Note          *string
}

// ==================== RESPONSE STRUCTS ====================

type CheckoutResponse struct {
	ID              string    `json:"id"`
	OrderNumber     string    `json:"orderNumber"`
	Status          string    `json:"status"`
	TotalPrice      float64   `json:"totalPrice"`
	PlacedAt        time.Time `json:"placedAt"`
	SnapToken       *string   `json:"snapToken,omitempty"`
	SnapRedirectUrl *string   `json:"snapRedirectUrl,omitempty"`
}

type OrderResponse struct {
	ID              string              `json:"id"`
	OrderNumber     string              `json:"orderNumber"`
	Status          string              `json:"status"`
	ReceiptNo       *string             `json:"receiptNo,omitempty"` // Tambahkan di sini
	PaymentStatus   string              `json:"paymentStatus"`
	SubtotalPrice   float64             `json:"subtotalPrice"`
	ShippingPrice   float64             `json:"shippingPrice"`
	TotalPrice      float64             `json:"totalPrice"`
	PlacedAt        time.Time           `json:"placedAt"`
	SnapToken       *string             `json:"snapToken,omitempty"`
	SnapRedirectUrl *string             `json:"snapRedirectUrl,omitempty"`
	Items           []OrderItemResponse `json:"items,omitempty"`
}

type OrderItemResponse struct {
	ID           string  `json:"id"`
	ProductID    string  `json:"productId"`
	NameSnapshot string  `json:"nameSnapshot"`
	UnitPrice    float64 `json:"unitPrice"`
	Quantity     int32   `json:"quantity"`
	Subtotal     float64 `json:"subtotal"` // unitPrice * quantity
}

type OrderDetailResponse struct {
	ID          string              `json:"id"`
	OrderNumber string              `json:"orderdNumber"`
	UserID      string              `json:"userId"`
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
	OrderNumber string    `json:"orderNumber"`
	UserID      string    `json:"userId"`
	UserEmail   string    `json:"userEmail,omitempty"` // jika perlu join dengan user table
	Status      string    `json:"status"`
	TotalPrice  float64   `json:"totalPrice"`
	PlacedAt    time.Time `json:"placedAt"`
}
