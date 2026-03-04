package customer

import (
	"context"
	"database/sql"
	"errors"
	"go-gadget-api/internal/address"
	"go-gadget-api/internal/order"
	"go-gadget-api/internal/pkg/response"
	"go-gadget-api/internal/shared/database/dbgen"
	"strconv"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

//go:generate mockgen -source=customer_service.go -destination=../mock/customer/customer_service_mock.go -package=mock
type Service interface {
	UpdateProfile(ctx context.Context, customerID string, req UpdateProfileRequest) (CustomerResponse, error)
	ListCustomers(ctx context.Context, page, limit int, search string) ([]CustomerListResponse, int64, error)
	ToggleCustomerStatus(ctx context.Context, customerID string, active bool) (CustomerListResponse, error)
	GetCustomerByID(ctx context.Context, req CustomerDetailsRequest) (CustomerDetailResponse, error)
	ListCustomerAddresses(ctx context.Context, req CustomerAddressesRequest) (PaginatedAddressResponse, error)
	ListCustomerOrders(ctx context.Context, req CustomerOrdersRequest) (PaginatedOrderResponse, error)
}

type service struct {
	repo        Repository
	addressRepo address.Repository
	orderRepo   order.Repository
	db          *sql.DB
}

func NewService(db *sql.DB, r Repository, ar address.Repository, or order.Repository) Service {
	return &service{
		db:          db,
		repo:        r,
		addressRepo: ar,
		orderRepo:   or,
	}
}

func (s *service) ListCustomers(ctx context.Context, page, limit int, search string) ([]CustomerListResponse, int64, error) {
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	searchNull := sql.NullString{String: search, Valid: search != ""}

	customers, err := s.repo.ListCustomers(ctx, dbgen.ListCustomersParams{
		Limit:  int32(limit),
		Offset: int32(offset),
		Search: searchNull,
	})
	if err != nil {
		return nil, 0, err
	}

	var total int64 = 0
	responses := make([]CustomerListResponse, 0, len(customers))
	for _, c := range customers {
		if total == 0 {
			total = c.TotalCount
		}
		responses = append(responses, CustomerListResponse{
			ID:        c.ID.String(),
			Name:      c.Name,
			Email:     c.Email,
			Phone:     c.Phone.String,
			IsActive:  c.IsActive,
			CreatedAt: c.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return responses, total, nil
}

func (s *service) ToggleCustomerStatus(ctx context.Context, customerID string, active bool) (CustomerListResponse, error) {
	id, err := uuid.Parse(customerID)
	if err != nil {
		return CustomerListResponse{}, err
	}

	res, err := s.repo.UpdateStatus(ctx, dbgen.UpdateCustomerStatusParams{
		ID:       id,
		IsActive: active,
	})
	if err != nil {
		return CustomerListResponse{}, err
	}

	return CustomerListResponse{
		ID:        res.ID.String(),
		Name:      res.Name,
		Email:     res.Email,
		Phone:     res.Phone.String,
		IsActive:  res.IsActive,
		CreatedAt: res.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *service) GetCustomerByID(ctx context.Context, req CustomerDetailsRequest) (CustomerDetailResponse, error) {
	uid, err := uuid.Parse(req.CustomerID)
	if err != nil {
		return CustomerDetailResponse{}, err
	}

	user, err := s.repo.GetByID(ctx, uid)
	if err != nil {
		return CustomerDetailResponse{}, ErrCustomerNotFound
	}

	return CustomerDetailResponse{
		ID:        user.ID.String(),
		Name:      user.Name,
		Email:     user.Email,
		Phone:     user.Phone.String,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *service) ListCustomerAddresses(ctx context.Context, req CustomerAddressesRequest) (PaginatedAddressResponse, error) {
	uid, err := uuid.Parse(req.CustomerID)
	if err != nil {
		return PaginatedAddressResponse{}, err
	}

	if req.Limit < 1 {
		req.Limit = 10
	}
	offset := (req.Page - 1) * req.Limit
	if offset < 0 {
		offset = 0
	}

	addrRows, err := s.addressRepo.ListByUser(ctx, dbgen.ListAddressesByUserParams{
		UserID: uid,
		Limit:  int32(req.Limit),
		Offset: int32(offset),
		Search: sql.NullString{String: req.Search, Valid: req.Search != ""},
	})
	if err != nil {
		return PaginatedAddressResponse{}, err
	}

	var addrTotal int64 = 0
	addressResponses := make([]AddressResponse, 0, len(addrRows))
	for _, a := range addrRows {
		if addrTotal == 0 {
			addrTotal = a.TotalCount
		}
		fullAddr := a.Street
		if a.Subdistrict.Valid {
			fullAddr += ", " + a.Subdistrict.String
		}
		if a.City.Valid {
			fullAddr += ", " + a.City.String
		}
		if a.Province.Valid {
			fullAddr += ", " + a.Province.String
		}

		addressResponses = append(addressResponses, AddressResponse{
			ID:           a.ID.String(),
			AddressName:  a.Label,
			ReceiverName: a.RecipientName,
			Phone:        a.RecipientPhone,
			FullAddress:  fullAddr,
			IsMain:       a.IsPrimary,
		})
	}

	return PaginatedAddressResponse{
		Data: addressResponses,
		Meta: response.NewPaginationMeta(addrTotal, req.Page, req.Limit),
	}, nil
}

func (s *service) ListCustomerOrders(ctx context.Context, req CustomerOrdersRequest) (PaginatedOrderResponse, error) {
	uid, err := uuid.Parse(req.CustomerID)
	if err != nil {
		return PaginatedOrderResponse{}, err
	}

	if req.Limit < 1 {
		req.Limit = 10
	}
	offset := (req.Page - 1) * req.Limit
	if offset < 0 {
		offset = 0
	}

	orderRows, err := s.orderRepo.List(ctx, dbgen.ListOrdersParams{
		UserID: uid,
		Limit:  int32(req.Limit),
		Offset: int32(offset),
		Search: sql.NullString{String: req.Search, Valid: req.Search != ""},
	})
	if err != nil {
		return PaginatedOrderResponse{}, err
	}

	var orderTotal int64 = 0
	orderResponses := make([]OrderResponse, 0, len(orderRows))
	for _, o := range orderRows {
		if orderTotal == 0 {
			orderTotal = o.TotalCount
		}
		totalPrice, _ := strconv.Atoi(o.TotalPrice)
		orderResponses = append(orderResponses, OrderResponse{
			ID:          o.ID.String(),
			OrderNumber: o.OrderNumber,
			TotalPrice:  totalPrice,
			Status:      o.Status,
			CreatedAt:   o.PlacedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return PaginatedOrderResponse{
		Data: orderResponses,
		Meta: response.NewPaginationMeta(orderTotal, req.Page, req.Limit),
	}, nil
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
