package brand

import "time"

// --- REQUEST DTO ---

type CreateBrandRequest struct {
	Name        string `json:"name" binding:"required" validate:"required,min=2,max=100"`
	Description string `json:"description" validate:"max=500"`
	ImageUrl    string `json:"image_url" validate:"omitempty,url"`
}

type UpdateBrandRequest struct {
	Name        string `json:"name" binding:"required" validate:"required,min=2,max=100"`
	Description string `json:"description" validate:"max=500"`
	ImageUrl    string `json:"image_url" validate:"omitempty,url"`
	IsActive    *bool  `json:"is_active" binding:"required" validate:"required"`
}

type ListBrandRequest struct {
	Page   int32  `form:"page"`
	Limit  int32  `form:"pageSize"`
	Search string `form:"search"`
	Sort   string `form:"sort"`
}

// --- RESPONSE DTO ---

type BrandPublicResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	ImageUrl    string `json:"image_url,omitempty"`
}

type BrandAdminResponse struct {
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
