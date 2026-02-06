package address

import (
	"context"
	"database/sql"
	"go-gadget-api/internal/dbgen"

	"github.com/google/uuid"
)

//go:generate mockgen -source=address_service.go -destination=../mock/address/address_service_mock.go -package=mock
type Service interface {
	List(ctx context.Context, userID string) ([]AddressResponse, error)
	Create(ctx context.Context, req CreateAddressRequest) (AddressResponse, error)
	Update(ctx context.Context, addressID string, userID string, req UpdateAddressRequest) (AddressResponse, error)
	Delete(ctx context.Context, addressID string, userID string) error
	ListAdmin(
		ctx context.Context,
		page int,
		limit int,
	) ([]AddressAdminResponse, int64, error)
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

func (s *service) List(ctx context.Context, userID string) ([]AddressResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	rows, err := s.repo.ListByUser(ctx, uid)
	if err != nil {
		return nil, err
	}

	var res []AddressResponse
	for _, r := range rows {
		res = append(res, mapRowToResponse(r))
	}

	return res, nil
}

func (s *service) Create(ctx context.Context, req CreateAddressRequest) (AddressResponse, error) {
	uid, _ := uuid.Parse(req.UserID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AddressResponse{}, err
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)

	if req.IsPrimary {
		if err := qtx.UnsetPrimaryByUser(ctx, uid); err != nil {
			return AddressResponse{}, err
		}
	}

	addr, err := qtx.Create(ctx, dbgen.CreateAddressParams{
		UserID:         uid,
		Label:          req.Label,
		RecipientName:  req.RecipientName,
		RecipientPhone: req.RecipientPhone,
		Street:         req.Street,
		Subdistrict:    dbgen.ToText(req.Subdistrict),
		District:       dbgen.ToText(req.District),
		City:           dbgen.ToText(req.City),
		Province:       dbgen.ToText(req.Province),
		PostalCode:     dbgen.ToText(req.PostalCode),
		IsPrimary:      req.IsPrimary,
	})
	if err != nil {
		return AddressResponse{}, err
	}

	if err := tx.Commit(); err != nil {
		return AddressResponse{}, err
	}

	return mapToResponse(addr), nil
}

func (s *service) Update(ctx context.Context, addressID string, userID string, req UpdateAddressRequest) (AddressResponse, error) {
	id, _ := uuid.Parse(addressID)
	uid, _ := uuid.Parse(userID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AddressResponse{}, err
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)

	if req.IsPrimary {
		if err := qtx.UnsetPrimaryByUser(ctx, uid); err != nil {
			return AddressResponse{}, err
		}
	}

	addr, err := qtx.Update(ctx, dbgen.UpdateAddressParams{
		ID:             id,
		Label:          req.Label,
		RecipientName:  req.RecipientName,
		RecipientPhone: req.RecipientPhone,
		Street:         req.Street,
		Subdistrict:    dbgen.ToText(req.Subdistrict),
		District:       dbgen.ToText(req.District),
		City:           dbgen.ToText(req.City),
		Province:       dbgen.ToText(req.Province),
		PostalCode:     dbgen.ToText(req.PostalCode),
		IsPrimary:      req.IsPrimary,
	})
	if err != nil {
		return AddressResponse{}, err
	}

	if err := tx.Commit(); err != nil {
		return AddressResponse{}, err
	}

	return mapToResponse(addr), nil
}

func (s *service) Delete(ctx context.Context, addressID string, userID string) error {
	id, _ := uuid.Parse(addressID)
	uid, _ := uuid.Parse(userID)
	return s.repo.Delete(ctx, id, uid)
}

func mapToResponse(a dbgen.Address) AddressResponse {
	return AddressResponse{
		ID:             a.ID.String(),
		Label:          a.Label,
		RecipientName:  a.RecipientName,
		RecipientPhone: a.RecipientPhone,
		Street:         a.Street,
		Subdistrict:    a.Subdistrict.String,
		District:       a.District.String,
		City:           a.City.String,
		Province:       a.Province.String,
		PostalCode:     a.PostalCode.String,
		IsPrimary:      a.IsPrimary,
		CreatedAt:      a.CreatedAt,
	}
}

func mapRowToResponse(r dbgen.ListAddressesByUserRow) AddressResponse {
	return AddressResponse{
		ID:             r.ID.String(),
		Label:          r.Label,
		RecipientName:  r.RecipientName,
		RecipientPhone: r.RecipientPhone,
		Street:         r.Street,
		Subdistrict:    r.Subdistrict.String,
		District:       r.District.String,
		City:           r.City.String,
		Province:       r.Province.String,
		PostalCode:     r.PostalCode.String,
		IsPrimary:      r.IsPrimary,
	}
}

func mapAdminRowToResponse(r dbgen.ListAddressesAdminRow) AddressAdminResponse {
	return AddressAdminResponse{
		ID:             r.ID.String(),
		UserID:         r.UserID.String(),
		UserEmail:      r.Email,
		Label:          r.Label,
		RecipientName:  r.RecipientName,
		RecipientPhone: r.RecipientPhone,
		Street:         r.Street,
		Subdistrict:    r.Subdistrict.String,
		District:       r.District.String,
		City:           r.City.String,
		Province:       r.Province.String,
		PostalCode:     r.PostalCode.String,
		IsPrimary:      r.IsPrimary,
	}
}

func (s *service) ListAdmin(
	ctx context.Context,
	page int,
	limit int,
) ([]AddressAdminResponse, int64, error) {

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	offset := int32((page - 1) * limit)

	rows, err := s.repo.ListAdmin(ctx, int32(limit), offset)
	if err != nil {
		return nil, 0, err
	}

	var (
		res   []AddressAdminResponse
		total int64
	)

	for _, r := range rows {
		total = r.TotalCount
		res = append(res, mapAdminRowToResponse(r))
	}

	return res, total, nil
}
