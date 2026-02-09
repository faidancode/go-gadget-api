package outbox

import (
	"context"
	"database/sql"
	"go-gadget-api/internal/shared/database/dbgen"

	"github.com/google/uuid"
)

//go:generate mockgen -source=outbox_repo.go -destination=../mock/outbox/outbox_repo_mock.go -package=mock
type Repository interface {
	WithTx(tx dbgen.DBTX) Repository
	CreateOutboxEvent(ctx context.Context, arg dbgen.CreateOutboxEventParams) error
	ListPending(ctx context.Context, limit int32) ([]dbgen.OutboxEvent, error)
	MarkSent(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID) error
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

func (r *repository) CreateOutboxEvent(
	ctx context.Context,
	arg dbgen.CreateOutboxEventParams,
) error {
	return r.queries.CreateOutboxEvent(ctx, arg)
}

func (r *repository) ListPending(
	ctx context.Context,
	limit int32,
) ([]dbgen.OutboxEvent, error) {
	return r.queries.ListPendingOutbox(ctx, limit)
}

func (r *repository) MarkSent(
	ctx context.Context,
	id uuid.UUID,
) error {
	return r.queries.MarkOutboxEventSent(ctx, id)
}

func (r *repository) MarkFailed(
	ctx context.Context,
	id uuid.UUID,
) error {
	return r.queries.MarkOutboxEventFailed(ctx, id)
}
