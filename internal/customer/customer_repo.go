package customer

import (
	"context"
	"database/sql"
	"go-gadget-api/internal/shared/database/dbgen"

	"github.com/google/uuid"
)

//go:generate mockgen -source=customer_repo.go -destination=../mock/customer/customer_repo_mock.go -package=mock
type Repository interface {
	WithTx(tx dbgen.DBTX) Repository

	GetByID(ctx context.Context, id uuid.UUID) (dbgen.GetUserByIDRow, error)
	UpdateProfile(ctx context.Context, arg dbgen.UpdateCustomerProfileParams) (dbgen.UpdateCustomerProfileRow, error)
	UpdatePassword(ctx context.Context, arg dbgen.UpdateCustomerPasswordParams) error
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

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (dbgen.GetUserByIDRow, error) {
	return r.queries.GetUserByID(ctx, id)
}

func (r *repository) UpdateProfile(
	ctx context.Context,
	arg dbgen.UpdateCustomerProfileParams,
) (dbgen.UpdateCustomerProfileRow, error) {
	return r.queries.UpdateCustomerProfile(ctx, arg)
}

func (r *repository) UpdatePassword(
	ctx context.Context,
	arg dbgen.UpdateCustomerPasswordParams,
) error {
	return r.queries.UpdateCustomerPassword(ctx, arg)
}
