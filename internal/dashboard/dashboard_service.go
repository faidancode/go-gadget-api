package dashboard

import (
	"context"
	"fmt"
	"go-gadget-api/internal/shared/database/dbgen"
	"strconv"
	"sync"
	"time"
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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error

		// Menggunakan tipe data dari dbgen
		stats        dbgen.GetDashboardStatsRow
		recentOrders []dbgen.ListRecentOrdersRow
		distribution []dbgen.GetCategoryDistributionRow
	)

	wg.Add(3)

	// --- Eksekusi Paralel (Goroutines) ---
	go func() {
		defer wg.Done()
		res, err := s.repo.GetStats(ctx)
		mu.Lock()
		if err != nil {
			errs = append(errs, err)
		}
		stats = res
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		res, err := s.repo.ListRecentOrders(ctx, 5) // limit int32
		mu.Lock()
		if err != nil {
			errs = append(errs, err)
		}
		recentOrders = res
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		res, err := s.repo.GetCategoryDistribution(ctx)
		mu.Lock()
		if err != nil {
			errs = append(errs, err)
		}
		distribution = res
		mu.Unlock()
	}()

	wg.Wait()

	if len(errs) > 0 {
		return DashboardResponse{}, errs[0]
	}

	// --- Final Mapping (Service Layer) ---
	// Gunakan fmt.Sprintf jika tipe data TotalRevenue dari sqlc adalah numeric/interface
	revenue, _ := strconv.ParseFloat(fmt.Sprintf("%v", stats.TotalRevenue), 64)

	return DashboardResponse{
		Stats: DashboardStatsResponse{
			TotalProducts:   stats.TotalProducts,
			TotalBrands:     stats.TotalBrands,
			TotalCategories: stats.TotalCategories,
			TotalCustomers:  stats.TotalCustomers,
			TotalOrders:     stats.TotalOrders,
			TotalRevenue:    revenue,
		},
		RecentOrders:         mapOrdersToDTO(recentOrders),
		CategoryDistribution: mapDistToDTO(distribution),
	}, nil
}

// --- Mappers (Private Helpers) ---

func mapOrdersToDTO(rows []dbgen.ListRecentOrdersRow) []RecentOrderResponse {
	res := make([]RecentOrderResponse, 0, len(rows))
	for _, r := range rows {
		total, _ := strconv.ParseFloat(r.TotalPrice, 64)
		res = append(res, RecentOrderResponse{
			ID:          r.ID.String(),
			OrderNumber: r.OrderNumber,
			TotalAmount: total,
			Status:      r.Status,
			Customer:    r.UserName,
			Date:        r.PlacedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return res
}

func mapDistToDTO(rows []dbgen.GetCategoryDistributionRow) []CategoryDistributionResponse {
	res := make([]CategoryDistributionResponse, 0, len(rows))
	for _, r := range rows {
		res = append(res, CategoryDistributionResponse{
			CategoryName: r.CategoryName,
			Count:        r.ProductCount,
		})
	}
	return res
}
