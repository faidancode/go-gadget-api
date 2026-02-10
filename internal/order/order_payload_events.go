package order

type DeleteCartPayload struct {
	UserID string `json:"user_id" validate:"required"`
}
