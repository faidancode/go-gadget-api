package cart

type AddItemRequest struct {
	ProductID string `json:"book_id" binding:"required"`
	Qty       int32  `json:"qty" binding:"required,min=1"`
	Price     int32  `json:"price_cents" binding:"required"`
}

type UpdateQtyRequest struct {
	Qty int32 `json:"qty" binding:"required,min=1"`
}

type CartCountResponse struct {
	Count int64 `json:"count"`
}

type CartItemDetailResponse struct {
	ID        string `json:"id"`
	ProductID string `json:"book_id"`
	Qty       int32  `json:"qty"`
	Price     int32  `json:"price_cents"`
	CreatedAt string `json:"created_at"`
}

type CartDetailResponse struct {
	Items []CartItemDetailResponse `json:"items"`
}
