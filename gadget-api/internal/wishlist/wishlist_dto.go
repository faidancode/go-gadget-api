package wishlist

import "time"

// ==================== REQUEST STRUCTS ====================

type AddItemRequest struct {
	ProductID string `json:"product_id" binding:"required"`
}

type DeleteItemRequest struct {
	ProductID string `json:"product_id" binding:"required"`
}

// ==================== RESPONSE STRUCTS ====================

type WishlistItemResponse struct {
	ID        string    `json:"id"`
	ProductID string    `json:"product_id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Stock     int32     `json:"stock"`
	ImageURL  string    `json:"image_url,omitempty"`
	AddedAt   time.Time `json:"added_at"`
}

type WishlistResponse struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Items     []WishlistItemResponse `json:"items"`
	ItemCount int                    `json:"item_count"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type AddItemResponse struct {
	Message string                `json:"message"`
	Item    *WishlistItemResponse `json:"item,omitempty"`
}
