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
