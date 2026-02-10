package customer

type UpdateProfileRequest struct {
	Name            *string `json:"name"`
	Password        *string `json:"password"`
	CurrentPassword *string `json:"current_password"`
}

type CustomerResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}
