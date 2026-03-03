package dashboard

import (
	"context"
	"fmt"
	"strconv"
)

//go:generate mockgen -source=dashboard_service.go -destination=../mock/dashboard/dashboard_service_mock.go -package=mock
type Service interface {
	GetDashboardData(ctx context.Context) (DashboardResponse, error)
}

type service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return &service{repo: r}
}

func (s *service) GetDashboardData(ctx context.Context) (DashboardResponse, error) {
	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		return DashboardResponse{}, err
	}

	recentOrders, err := s.repo.ListRecentOrders(ctx, 5)
	if err != nil {
		return DashboardResponse{}, err
	}

	distribution, err := s.repo.GetCategoryDistribution(ctx)
	if err != nil {
		return DashboardResponse{}, err
	}

	revenue, _ := strconv.ParseFloat(fmt.Sprintf("%v", stats.TotalRevenue), 64)

	recentOrderResponses := make([]RecentOrderResponse, 0, len(recentOrders))
	for _, o := range recentOrders {
		total, _ := strconv.ParseFloat(o.TotalPrice, 64)
		recentOrderResponses = append(recentOrderResponses, RecentOrderResponse{
			ID:          o.ID.String(),
			OrderNumber: o.OrderNumber,
			TotalAmount: total,
			Status:      o.Status,
			Customer:    o.UserName,
			Date:        o.PlacedAt.Format("2006-01-02 15:04:05"),
		})
	}

	distResponses := make([]CategoryDistributionResponse, 0, len(distribution))
	for _, d := range distribution {
		distResponses = append(distResponses, CategoryDistributionResponse{
			CategoryName: d.CategoryName,
			Count:        d.ProductCount,
		})
	}

	return DashboardResponse{
		Stats: DashboardStatsResponse{
			TotalProducts:   stats.TotalProducts,
			TotalBrands:     stats.TotalBrands,
			TotalCategories: stats.TotalCategories,
			TotalCustomers:  stats.TotalCustomers,
			TotalOrders:     stats.TotalOrders,
			TotalRevenue:    revenue,
		},
		RecentOrders:         recentOrderResponses,
		CategoryDistribution: distResponses,
	}, nil
}
