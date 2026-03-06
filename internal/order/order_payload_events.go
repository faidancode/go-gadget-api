package order

type DeleteCartPayload struct {
	UserID string `json:"user_id" validate:"required"`
}

type OrderStatusChangedPayload struct {
	OrderID     string `json:"order_id"`
	OrderNumber string `json:"order_number"`
	UserID      string `json:"user_id"`
	OldStatus   string `json:"old_status"`
	NewStatus   string `json:"new_status"`
	ChangedAt   string `json:"changed_at"`
}

type OrderPaymentUpdatedPayload struct {
	OrderID     string `json:"order_id"`
	OrderNumber string `json:"order_number"`
	UserID      string `json:"user_id"`
	OldStatus   string `json:"old_status"`
	NewStatus   string `json:"new_status"`
	ChangedAt   string `json:"changed_at"`
}
