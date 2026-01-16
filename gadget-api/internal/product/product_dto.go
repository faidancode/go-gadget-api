package product

import "time"

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

// ProductResponse adalah output ringkas untuk Customer
type ProductPublicResponse struct {
	ID           string  `json:"id"`
	CategoryName string  `json:"category_name"`
	Name         string  `json:"name"`
	Slug         string  `json:"slug"`
	Price        float64 `json:"price"`
}

// ProductAdminResponse adalah output detail untuk Dashboard Admin (Ini yang menyebabkan error)
type ProductAdminResponse struct {
	ID           string    `json:"id"`
	CategoryName string    `json:"category_name"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Price        float64   `json:"price"`
	Stock        int32     `json:"stock"`
	SKU          string    `json:"sku"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
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
