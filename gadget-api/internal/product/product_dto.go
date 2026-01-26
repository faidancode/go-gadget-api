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

type CreateProductRequest struct {
	CategoryID  string  `json:"categoryId" validate:"required"`
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" validate:"required,gt=0"`
	Stock       int32   `json:"stock" validate:"required,min=0"`
	SKU         string  `json:"sku"`
	ImageUrl    string  `json:"imageUrl"`
}

type UpdateProductRequest struct {
	CategoryID  string  `json:"categoryId"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price" validate:"omitempty,gt=0"`
	Stock       int32   `json:"stock" validate:"omitempty,min=0"`
	SKU         string  `json:"sku"`
	ImageUrl    string  `json:"imageUrl"`
	IsActive    *bool   `json:"isActive"` // Tetap menggunakan pointer untuk opsionalitas
}

// ==================== RESPONSE STRUCTS ====================

// ProductPublicResponse untuk list produk (ringkas)
type ProductPublicResponse struct {
	ID           string  `json:"id"`
	CategoryName string  `json:"categoryName"`
	Name         string  `json:"name"`
	Slug         string  `json:"slug"`
	Price        float64 `json:"price"`
	ImageURL     string  `json:"imageUrl,omitempty"`
}

// ProductDetailResponse untuk detail produk dengan reviews
type ProductDetailResponse struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Slug           string            `json:"slug"`
	Description    string            `json:"description"`
	Price          float64           `json:"price"`
	Stock          int32             `json:"stock"`
	CategoryID     string            `json:"categoryId,omitempty"`
	CategoryName   string            `json:"categoryName,omitempty"`
	BrandID        string            `json:"brandId,omitempty"`
	BrandName      string            `json:"brandName,omitempty"`
	ImageURL       string            `json:"imageUrl,omitempty"`
	SKU            string            `json:"sku,omitempty"`
	Specifications map[string]string `json:"specifications,omitempty"`

	// Review fields
	Reviews       []ReviewSummary `json:"reviews"`
	AverageRating float64         `json:"averageRating"`
	RatingCount   int64           `json:"ratingCount"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ReviewSummary for product detail (5 reviews terbaru)
type ReviewSummary struct {
	ID        string    `json:"id"`
	UserName  string    `json:"userName"`
	Rating    int32     `json:"rating"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"createdAt"`
}

// ProductAdminResponse untuk dashboard admin
type ProductAdminResponse struct {
	ID           string    `json:"id"`
	CategoryName string    `json:"categoryName"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Price        float64   `json:"price"`
	Stock        int32     `json:"stock"`
	SKU          string    `json:"sku"`
	ImageURL     string    `json:"imageUrl,omitempty"`
	IsActive     bool      `json:"isActive"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
