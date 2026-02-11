package address

import "time"

// ========= REQUEST =========

type CreateAddressRequest struct {
	UserID         string `json:"userId" binding:"required"`
	Label          string `json:"label" binding:"required"`
	RecipientName  string `json:"recipientName" binding:"required"`
	RecipientPhone string `json:"recipientPhone" binding:"required"`
	Street         string `json:"street" binding:"required"`
	Subdistrict    string `json:"subdistrict"`
	District       string `json:"district"`
	City           string `json:"city"`
	Province       string `json:"province"`
	PostalCode     string `json:"postalCode"`
	IsPrimary      bool   `json:"isPrimary"`
}

type UpdateAddressRequest struct {
	Label          string `json:"label" binding:"required"`
	RecipientName  string `json:"recipientName" binding:"required"`
	RecipientPhone string `json:"recipientPhone" binding:"required"`
	Street         string `json:"street" binding:"required"`
	Subdistrict    string `json:"subdistrict"`
	District       string `json:"district"`
	City           string `json:"city"`
	Province       string `json:"province"`
	PostalCode     string `json:"postalCode"`
	IsPrimary      bool   `json:"isPrimary"`
}

// ========= RESPONSE =========

type AddressResponse struct {
	ID             string    `json:"id"`
	Label          string    `json:"label"`
	RecipientName  string    `json:"recipientName"`
	RecipientPhone string    `json:"recipientPhone"`
	Street         string    `json:"street"`
	Subdistrict    string    `json:"subdistrict,omitempty"`
	District       string    `json:"district,omitempty"`
	City           string    `json:"city,omitempty"`
	Province       string    `json:"province,omitempty"`
	PostalCode     string    `json:"postalCode,omitempty"`
	IsPrimary      bool      `json:"isPrimary"`
	CreatedAt      time.Time `json:"createdAt"`
}

type AddressAdminResponse struct {
	ID             string `json:"id"`
	UserID         string `json:"userId"`
	UserEmail      string `json:"userEmail"`
	Label          string `json:"label"`
	RecipientName  string `json:"recipientName"`
	RecipientPhone string `json:"recipientPhone"`
	Street         string `json:"street"`
	Subdistrict    string `json:"subdistrict,omitempty"`
	District       string `json:"district,omitempty"`
	City           string `json:"city,omitempty"`
	Province       string `json:"province,omitempty"`
	PostalCode     string `json:"postalCode,omitempty"`
	IsPrimary      bool   `json:"isPrimary"`
	CreatedAt      string `json:"createdAt"`
}
