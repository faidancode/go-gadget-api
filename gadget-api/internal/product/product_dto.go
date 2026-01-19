package product

import "time"

// ==================== REQUEST STRUCTS ====================

// ListPublicRequest digunakan untuk menampung query params dari Customer
type ListPublicRequest struct {
	Page       int
	Limit      int
	Search     string
	CategoryID string
	MinPrice   float64
	MaxPrice   float64
	SortBy     string
}

type ListProductAdminRequest struct {
	Page     int
	Limit    int
	Search   string
	Category string
	SortBy   string
	SortDir  string // asc | desc
}

// CreateProductRequest digunakan untuk input Admin saat membuat produk baru
type CreateProductRequest struct {
	CategoryID  string  `json:"category_id" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required"`
	Stock       int32   `json:"stock" binding:"required"`
	SKU         string  `json:"sku"`
	ImageUrl    string  `json:"image_url"`
}

type UpdateProductRequest struct {
	CategoryID  string  `json:"category_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int32   `json:"stock"`
	SKU         string  `json:"sku"`
	ImageUrl    string  `json:"image_url"`
	IsActive    *bool   `json:"is_active"` // Gunakan pointer agar bisa membedakan false (bool) dan nil (tidak dikirim)
}

// ==================== RESPONSE STRUCTS ====================

// ProductPublicResponse untuk list produk (ringkas)
type ProductPublicResponse struct {
	ID           string  `json:"id"`
	CategoryName string  `json:"category_name"`
	Name         string  `json:"name"`
	Slug         string  `json:"slug"`
	Price        float64 `json:"price"`
	ImageURL     string  `json:"image_url,omitempty"`
}

// ProductDetailResponse untuk detail produk dengan reviews
type ProductDetailResponse struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Slug           string            `json:"slug"`
	Description    string            `json:"description"`
	Price          float64           `json:"price"`
	Stock          int32             `json:"stock"`
	CategoryID     string            `json:"category_id,omitempty"`
	CategoryName   string            `json:"category_name,omitempty"`
	BrandID        string            `json:"brand_id,omitempty"`
	BrandName      string            `json:"brand_name,omitempty"`
	ImageURL       string            `json:"image_url,omitempty"`
	SKU            string            `json:"sku,omitempty"`
	Specifications map[string]string `json:"specifications,omitempty"`

	// Review fields
	Reviews       []ReviewSummary `json:"reviews"`
	AverageRating float64         `json:"average_rating"`
	TotalReviews  int64           `json:"total_reviews"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ReviewSummary for product detail (5 reviews terbaru)
type ReviewSummary struct {
	ID        string    `json:"id"`
	UserName  string    `json:"user_name"`
	Rating    int32     `json:"rating"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

// ProductAdminResponse untuk dashboard admin
type ProductAdminResponse struct {
	ID           string    `json:"id"`
	CategoryName string    `json:"category_name"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Price        float64   `json:"price"`
	Stock        int32     `json:"stock"`
	SKU          string    `json:"sku"`
	ImageURL     string    `json:"image_url,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
