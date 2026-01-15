package category

import "time"

type CreateCategoryRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Description string `json:"description" validate:"max=500"`
	ImageUrl    string `json:"image_url" validate:"omitempty,url"`
}

type CategoryResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
