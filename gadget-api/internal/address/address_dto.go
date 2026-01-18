package address

import "time"

// ========= REQUEST =========

type CreateAddressRequest struct {
	UserID         string `json:"-"`
	Label          string `json:"label" binding:"required"`
	RecipientName  string `json:"recipient_name" binding:"required"`
	RecipientPhone string `json:"recipient_phone" binding:"required"`
	Street         string `json:"street" binding:"required"`
	Subdistrict    string `json:"subdistrict"`
	District       string `json:"district"`
	City           string `json:"city"`
	Province       string `json:"province"`
	PostalCode     string `json:"postal_code"`
	IsPrimary      bool   `json:"is_primary"`
}

type UpdateAddressRequest struct {
	Label          string `json:"label" binding:"required"`
	RecipientName  string `json:"recipient_name" binding:"required"`
	RecipientPhone string `json:"recipient_phone" binding:"required"`
	Street         string `json:"street" binding:"required"`
	Subdistrict    string `json:"subdistrict"`
	District       string `json:"district"`
	City           string `json:"city"`
	Province       string `json:"province"`
	PostalCode     string `json:"postal_code"`
	IsPrimary      bool   `json:"is_primary"`
}

// ========= RESPONSE =========

type AddressResponse struct {
	ID             string    `json:"id"`
	Label          string    `json:"label"`
	RecipientName  string    `json:"recipient_name"`
	RecipientPhone string    `json:"recipient_phone"`
	Street         string    `json:"street"`
	Subdistrict    string    `json:"subdistrict,omitempty"`
	District       string    `json:"district,omitempty"`
	City           string    `json:"city,omitempty"`
	Province       string    `json:"province,omitempty"`
	PostalCode     string    `json:"postal_code,omitempty"`
	IsPrimary      bool      `json:"is_primary"`
	CreatedAt      time.Time `json:"created_at"`
}

type AddressAdminResponse struct {
	ID             string `json:"id"`
	UserID         string `json:"user_id"`
	UserEmail      string `json:"user_email"`
	Label          string `json:"label"`
	RecipientName  string `json:"recipient_name"`
	RecipientPhone string `json:"recipient_phone"`
	Street         string `json:"street"`
	Subdistrict    string `json:"subdistrict,omitempty"`
	District       string `json:"district,omitempty"`
	City           string `json:"city,omitempty"`
	Province       string `json:"province,omitempty"`
	PostalCode     string `json:"postal_code,omitempty"`
	IsPrimary      bool   `json:"is_primary"`
	CreatedAt      string `json:"created_at"`
}
