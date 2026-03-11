package dashboard_test

import (
	"context"
	"errors"
	"go-gadget-api/internal/dashboard"
	mock "go-gadget-api/internal/mock/dashboard"
	"go-gadget-api/internal/shared/database/dbgen"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDashboardService_GetDashboardData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	svc := dashboard.NewService(repo)

	// Kita tetap pakai ini untuk input awal
	ctx := context.Background()

	t.Run("success_fetch_dashboard_data", func(t *testing.T) {
		// PERBAIKAN: Gunakan gomock.Any() karena service akan membungkus ctx dengan Timeout
		repo.EXPECT().
			GetStats(gomock.Any()).
			Return(dbgen.GetDashboardStatsRow{
				TotalProducts:   10,
				TotalBrands:     5,
				TotalCategories: 3,
				TotalCustomers:  20,
				TotalOrders:     50,
				TotalRevenue:    1000000,
			}, nil)

		repo.EXPECT().
			ListRecentOrders(gomock.Any(), int32(5)).
			Return([]dbgen.ListRecentOrdersRow{
				{
					ID:          uuid.New(),
					OrderNumber: "ORD-001",
					TotalPrice:  "100000",
					Status:      "COMPLETED",
					UserName:    "John Doe",
					PlacedAt:    time.Now(),
				},
			}, nil)

		repo.EXPECT().
			GetCategoryDistribution(gomock.Any()).
			Return([]dbgen.GetCategoryDistributionRow{
				{
					CategoryName: "Category A",
					ProductCount: 5,
				},
			}, nil)

		res, err := svc.GetDashboardData(ctx)

		assert.NoError(t, err)
		assert.Equal(t, int64(10), res.Stats.TotalProducts)
		assert.Len(t, res.RecentOrders, 1)
		assert.Equal(t, "John Doe", res.RecentOrders[0].Customer)
		assert.Len(t, res.CategoryDistribution, 1)
	})

	t.Run("error_repository_failure", func(t *testing.T) {
		// Gunakan gomock.Any() juga di sini
		repo.EXPECT().GetStats(gomock.Any()).Return(dbgen.GetDashboardStatsRow{}, errors.New("db error"))

		// Karena kita menggunakan goroutine paralel, mock lain mungkin terpanggil
		// atau tidak tergantung mana yang duluan kena error.
		// Untuk amannya di test error, kita bisa gunakan .AnyTimes() atau
		// fokus pada error return-nya saja.
		repo.EXPECT().ListRecentOrders(gomock.Any(), gomock.Any()).AnyTimes()
		repo.EXPECT().GetCategoryDistribution(gomock.Any()).AnyTimes()

		_, err := svc.GetDashboardData(ctx)
		assert.Error(t, err)
	})
}
