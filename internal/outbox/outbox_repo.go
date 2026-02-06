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
}

type outboxRepository struct {
	queries *dbgen.Queries
}

func NewRepository(q *dbgen.Queries) Repository {
	return &outboxRepository{queries: q}
}

func (r *outboxRepository) WithTx(tx dbgen.DBTX) Repository {
	if sqlTx, ok := tx.(*sql.Tx); ok {
		return &outboxRepository{
			queries: r.queries.WithTx(sqlTx),
		}
	}
	return r
}

func (r *outboxRepository) CreateOutboxEvent(
	ctx context.Context,
	arg dbgen.CreateOutboxEventParams,
) error {
	return r.queries.CreateOutboxEvent(ctx, arg)
}

func (r *outboxRepository) ListPending(
	ctx context.Context,
	limit int32,
) ([]dbgen.OutboxEvent, error) {
	return r.queries.ListPendingOutbox(ctx, limit)
}

func (r *outboxRepository) MarkSent(
	ctx context.Context,
	id uuid.UUID,
) error {
	return r.queries.MarkOutboxSent(ctx, id)
}
