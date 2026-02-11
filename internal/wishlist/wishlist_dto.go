package wishlist

import "time"

// ==================== REQUEST STRUCTS ====================

type AddItemRequest struct {
	ProductID string `json:"productId" binding:"required"`
}

type DeleteItemRequest struct {
	ProductID string `json:"productId" binding:"required"`
}

// ==================== RESPONSE STRUCTS ====================

type WishlistItemResponse struct {
	ID      string                  `json:"id"`
	Product WishlistProductResponse `json:"product"`
}

type WishlistProductResponse struct {
	ID            string   `json:"id"`
	Slug          string   `json:"slug"`
	Name          string   `json:"name"`
	CategoryName  string   `json:"categoryName"`
	Price         float64  `json:"price"`                   // cents
	DiscountPrice *float64 `json:"discountPrice,omitempty"` // cents
	Stock         int32    `json:"stock"`
	ImageURL      *string  `json:"imageUrl,omitempty"`
}

type WishlistResponse struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"userId"`
	Items     []WishlistItemResponse `json:"items"`
	ItemCount int                    `json:"itemCount"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

type AddItemResponse struct {
	Message string                `json:"message"`
	Item    *WishlistItemResponse `json:"item,omitempty"`
}
