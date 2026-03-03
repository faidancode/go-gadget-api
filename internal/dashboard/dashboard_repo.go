package dashboard

import (
	"context"
	"go-gadget-api/internal/shared/database/dbgen"
)

//go:generate mockgen -source=dashboard_repo.go -destination=../mock/dashboard/dashboard_repo_mock.go -package=mock
type Repository interface {
	GetStats(ctx context.Context) (dbgen.GetDashboardStatsRow, error)
	ListRecentOrders(ctx context.Context, limit int32) ([]dbgen.ListRecentOrdersRow, error)
	GetCategoryDistribution(ctx context.Context) ([]dbgen.GetCategoryDistributionRow, error)
}

type repository struct {
	queries *dbgen.Queries
}

func NewRepository(q *dbgen.Queries) Repository {
	return &repository{queries: q}
}

func (r *repository) GetStats(ctx context.Context) (dbgen.GetDashboardStatsRow, error) {
	return r.queries.GetDashboardStats(ctx)
}

func (r *repository) ListRecentOrders(ctx context.Context, limit int32) ([]dbgen.ListRecentOrdersRow, error) {
	return r.queries.ListRecentOrders(ctx, limit)
}

func (r *repository) GetCategoryDistribution(ctx context.Context) ([]dbgen.GetCategoryDistributionRow, error) {
	return r.queries.GetCategoryDistribution(ctx)
}
