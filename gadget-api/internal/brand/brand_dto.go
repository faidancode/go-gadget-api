package brand

import "time"

// --- REQUEST DTO ---

type CreateBrandRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Slug        string `json:"slug"`
	Description string `json:"description" validate:"max=500"`
	ImageUrl    string `json:"imageUrl" validate:"omitempty,url"`
}

type UpdateBrandRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Slug        string `json:"slug"`
	Description string `json:"description" validate:"max=500"`
	ImageUrl    string `json:"imageUrl" validate:"omitempty,url"`
	IsActive    *bool  `json:"isActive"`
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
	ImageUrl    string `json:"imageUrl,omitempty"`
}

type BrandAdminResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description"`
	ImageUrl    string     `json:"imageUrl"`
	IsActive    bool       `json:"isActive"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}
