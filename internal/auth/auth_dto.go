package auth

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required"`
	Phone    string `json:"phone" binding:"required,numeric,min=10,max=15"`
	Password string `json:"password" binding:"required,min=8,max=32"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Role  string `json:"role"`
}

type RequestPasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type RequestEmailConfirmationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ConfirmEmailByPinRequest struct {
	Email string `json:"email" binding:"required,email"`
	PIN   string `json:"pin" binding:"required,len=6"`
}

type ActionStatusResponse struct {
	Success   bool   `json:"success"`
	EmailSent bool   `json:"emailSent,omitempty"`
	Message   string `json:"message,omitempty"`
}
