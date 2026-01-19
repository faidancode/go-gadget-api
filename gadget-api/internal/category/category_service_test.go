package category_test

import (
	"context"
	"database/sql"
	"errors"
	"gadget-api/internal/category"
	"gadget-api/internal/dbgen"
	categoryMock "gadget-api/internal/mock/category"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func dummyCategoryRow(total int64) dbgen.ListCategoriesRow {
	return dbgen.ListCategoriesRow{
		ID:         uuid.New(),
		Name:       "Laptop",
		Slug:       "laptop",
		TotalCount: total,

		// field lain boleh default / zero value
		Description: sql.NullString{},
		ImageUrl:    sql.NullString{},
		IsActive:    sql.NullBool{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func TestService_Category(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := categoryMock.NewMockRepository(ctrl)
	service := category.NewService(mockRepo)
	ctx := context.Background()

	// Helper data
	id := uuid.New()
	idStr := id.String()
	dummyCat := dbgen.Category{ID: id, Name: "Smartphone", Slug: "smartphone"}

	// 1. CREATE
	t.Run("Create - Success", func(t *testing.T) {
		req := category.CreateCategoryRequest{Name: "Smartphone"}
		mockRepo.EXPECT().Create(ctx, gomock.Any()).Return(dummyCat, nil)

		res, err := service.Create(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, "Smartphone", res.Name)
	})

	// 2. GET ALL (SINKRON: page, limit & EXPECT().List)
	t.Run("GetAll - Success", func(t *testing.T) {
		mockRepo.EXPECT().
			List(ctx, int32(10), int32(0)).
			Return([]dbgen.ListCategoriesRow{
				dummyCategoryRow(1),
			}, nil)

		res, total, err := service.GetAll(ctx, 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, res, 1)
		assert.Equal(t, "Laptop", res[0].Name)
	})

	// 3. GET BY ID (SINKRON: string idStr)
	t.Run("GetByID - Success", func(t *testing.T) {
		mockRepo.EXPECT().
			GetByID(ctx, id).
			Return(dummyCat, nil)

		res, err := service.GetByID(ctx, idStr) // idStr = id.String()
		assert.NoError(t, err)

		// FIX: bandingkan string dengan string
		assert.Equal(t, id.String(), res.ID)
	})

	t.Run("GetByID - Not Found", func(t *testing.T) {
		mockRepo.EXPECT().GetByID(ctx, id).Return(dbgen.Category{}, errors.New("not found"))

		_, err := service.GetByID(ctx, idStr)
		assert.Error(t, err)
	})

	// 4. UPDATE
	t.Run("Update - Success", func(t *testing.T) {
		req := category.CreateCategoryRequest{Name: "Updated Name"}
		mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(dbgen.Category{ID: id, Name: "Updated Name"}, nil)

		res, err := service.Update(ctx, idStr, req) // idStr string
		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", res.Name)
	})

	// 5. DELETE
	t.Run("Delete - Success", func(t *testing.T) {
		mockRepo.EXPECT().Delete(ctx, id).Return(nil)

		err := service.Delete(ctx, idStr) // idStr string
		assert.NoError(t, err)
	})
}
