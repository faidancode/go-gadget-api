package customer

import (
	"context"
	"database/sql"
	"errors"
	"go-gadget-api/internal/shared/database/dbgen"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

//go:generate mockgen -source=customer_service.go -destination=../mock/customer/customer_service_mock.go -package=mock
type Service interface {
	UpdateProfile(ctx context.Context, customerID string, req UpdateProfileRequest) (CustomerResponse, error)
}

type service struct {
	repo Repository
	db   *sql.DB
}

func NewService(db *sql.DB, r Repository) Service {
	return &service{
		db:   db,
		repo: r,
	}
}

func (s *service) UpdateProfile(
	ctx context.Context,
	customerID string,
	req UpdateProfileRequest,
) (CustomerResponse, error) {

	id, err := uuid.Parse(customerID)
	if err != nil {
		return CustomerResponse{}, err
	}

	// 1. Ambil user existing (untuk validasi password)
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return CustomerResponse{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CustomerResponse{}, err
	}
	defer tx.Rollback()

	repoTx := s.repo.WithTx(tx)

	var updatedUser *dbgen.UpdateCustomerProfileRow

	// 2. Update name (kalau ada)
	if req.Name != nil {
		res, err := repoTx.UpdateProfile(ctx, dbgen.UpdateCustomerProfileParams{
			ID:   id,
			Name: *req.Name,
		})
		if err != nil {
			return CustomerResponse{}, err
		}
		updatedUser = &res
	}

	// 3. Update password (kalau ada)
	if req.Password != nil && *req.Password != "" {
		if req.CurrentPassword == nil || *req.CurrentPassword == "" {
			return CustomerResponse{}, errors.New("current password is required")
		}

		if err := bcrypt.CompareHashAndPassword(
			[]byte(user.Password),
			[]byte(*req.CurrentPassword),
		); err != nil {
			return CustomerResponse{}, errors.New("invalid current password")
		}

		hashed, err := bcrypt.GenerateFromPassword(
			[]byte(*req.Password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			return CustomerResponse{}, err
		}

		if err := repoTx.UpdatePassword(ctx, dbgen.UpdateCustomerPasswordParams{
			ID:       id,
			Password: string(hashed),
		}); err != nil {
			return CustomerResponse{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return CustomerResponse{}, err
	}

	// 4. Response source
	if updatedUser != nil {
		return ToCustomerResponseFromProfile(*updatedUser), nil
	}

	// fallback (kalau cuma update password)
	return ToCustomerResponseFromUser(user), nil
}

func ToCustomerResponseFromUser(u dbgen.GetUserByIDRow) CustomerResponse {
	return CustomerResponse{
		ID:    u.ID.String(),
		Name:  u.Name,
		Email: u.Email,
	}
}

func ToCustomerResponseFromProfile(u dbgen.UpdateCustomerProfileRow) CustomerResponse {
	return CustomerResponse{
		ID:    u.ID.String(),
		Name:  u.Name,
		Email: u.Email,
	}
}
