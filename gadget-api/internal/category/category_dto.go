package category

import "time"

// --- REQUEST DTO ---

type CreateCategoryRequest struct {
	// Tambahkan binding:"required" agar Gin menjalankan validasi otomatis
	Name        string `json:"name" binding:"required" validate:"required,min=2,max=100"`
	Description string `json:"description" validate:"max=500"`
	ImageUrl    string `json:"image_url" validate:"omitempty,url"`
}

type UpdateCategoryRequest struct {
	Name        string `json:"name" binding:"required" validate:"required,min=2,max=100"`
	Description string `json:"description" validate:"max=500"`
	ImageUrl    string `json:"image_url" validate:"omitempty,url"`
	// Menggunakan *bool agar bisa membedakan antara false (dikirim) dan nil (tidak dikirim)
	IsActive *bool `json:"is_active" binding:"required" validate:"required"`
}

type ListCategoryRequest struct {
	// Gunakan tag form untuk binding dari query params URL
	Page    int    `form:"page" binding:"omitempty,min=1"`
	Limit   int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Search  string `form:"search"`
	SortBy  string `form:"sort_by" binding:"omitempty,oneof=name created_at"`
	SortDir string `form:"sort_dir" binding:"omitempty,oneof=asc desc"`
}

// --- RESPONSE DTO ---

type CategoryPublicResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	ImageUrl    string `json:"image_url,omitempty"`
}

type CategoryAdminResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description"`
	ImageUrl    string     `json:"image_url"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}
