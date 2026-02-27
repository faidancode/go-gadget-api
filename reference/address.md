## Repo

package address

import (
	"context"
	"database/sql"
	"go-gadget-api/internal/shared/database/dbgen"

	"github.com/google/uuid"
)

//go:generate mockgen -source=address_repo.go -destination=../mock/address/address_repo_mock.go -package=mock
type Repository interface {
	WithTx(tx dbgen.DBTX) Repository
	ListByUser(ctx context.Context, userID uuid.UUID) ([]dbgen.ListAddressesByUserRow, error)
	Create(ctx context.Context, arg dbgen.CreateAddressParams) (dbgen.Address, error)
	Update(ctx context.Context, arg dbgen.UpdateAddressParams) (dbgen.Address, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	UnsetPrimaryByUser(ctx context.Context, userID uuid.UUID) error

	ListAdmin(
		ctx context.Context,
		limit int32,
		offset int32,
	) ([]dbgen.ListAddressesAdminRow, error)
}

type repository struct {
	queries *dbgen.Queries
}

func NewRepository(q *dbgen.Queries) Repository {
	return &repository{queries: q}
}

func (r *repository) WithTx(tx dbgen.DBTX) Repository {
	if sqlTx, ok := tx.(*sql.Tx); ok {
		return &repository{
			queries: r.queries.WithTx(sqlTx),
		}
	}
	return r
}

func (r *repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]dbgen.ListAddressesByUserRow, error) {
	return r.queries.ListAddressesByUser(ctx, userID)
}

func (r *repository) Create(ctx context.Context, arg dbgen.CreateAddressParams) (dbgen.Address, error) {
	return r.queries.CreateAddress(ctx, arg)
}

func (r *repository) Update(ctx context.Context, arg dbgen.UpdateAddressParams) (dbgen.Address, error) {
	return r.queries.UpdateAddress(ctx, arg)
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.queries.SoftDeleteAddress(ctx, dbgen.SoftDeleteAddressParams{
		ID:     id,
		UserID: userID,
	})
}

func (r *repository) UnsetPrimaryByUser(ctx context.Context, userID uuid.UUID) error {
	return r.queries.UnsetPrimaryAddressByUser(ctx, userID)
}

func (r *repository) ListAdmin(
	ctx context.Context,
	limit int32,
	offset int32,
) ([]dbgen.ListAddressesAdminRow, error) {

	return r.queries.ListAddressesAdmin(
		ctx,
		dbgen.ListAddressesAdminParams{
			Limit:  limit,
			Offset: offset,
		},
	)
}


## Service

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


## Handler


type Handler struct {
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

// GET /addresses
func (h *Handler) List(c *gin.Context) {
	userID := c.GetString("user_id")

	res, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FAILED", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// POST /addresses
func (h *Handler) Create(c *gin.Context) {
	userID := c.GetString("user_id")

	var req CreateAddressRequest
	req.UserID = userID
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
		return
	}

	res, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		log.Println("ERROR CREATE ADDRESS REQ:", req)
		log.Println("ERROR CREATE ADDRESS:", err)
		response.Error(c, http.StatusInternalServerError, "FAILED", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusCreated, res, nil)
}

// PUT /addresses/:id
func (h *Handler) Update(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	var req UpdateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
		return
	}

	res, err := h.service.Update(c.Request.Context(), id, userID, req)
	if err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}