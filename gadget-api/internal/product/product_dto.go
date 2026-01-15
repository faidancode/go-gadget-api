package product

import "time"

type CreateProductRequest struct {
	CategoryID  string  `json:"category_id" binding:"required"`
	Name        string  `json:"name" binding:"required,min=2"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required,min=0"`
	Stock       int32   `json:"stock" binding:"required,min=0"`
	SKU         string  `json:"sku"`
	ImageUrl    string  `json:"image_url"`
}

type ProductResponse struct {
	ID           string    `json:"id"`
	CategoryName string    `json:"category_name,omitempty"`
	CategoryID   string    `json:"category_id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Price        float64   `json:"price"`
	Stock        int32     `json:"stock"`
	SKU          string    `json:"sku"`
	CreatedAt    time.Time `json:"created_at"`
}
