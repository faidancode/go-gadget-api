package cart

type AddItemRequest struct {
	ProductID string `json:"productId" validate:"required"`
	Qty       int32  `json:"qty" validate:"required,min=1"`
	Price     int32  `json:"price" validate:"required"`
}

type UpdateQtyRequest struct {
	Qty int32 `json:"qty" validate:"required,min=1"`
}

type CartCountResponse struct {
	Count int64 `json:"count"`
}

type CartItemDetailResponse struct {
	ID        string `json:"id"`
	ProductID string `json:"productId"`
	Qty       int32  `json:"qty"`
	Price     int32  `json:"price"`
	CreatedAt string `json:"createdAt"`
}

type CartDetailResponse struct {
	Items []CartItemDetailResponse `json:"items"`
}
