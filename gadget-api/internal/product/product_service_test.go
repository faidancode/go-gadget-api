package product

import (
	"context"
	"errors"
	catMock "gadget-api/internal/category/mock"
	"gadget-api/internal/dbgen"
	"gadget-api/internal/product/mock"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ========================
// CREATE PRODUCT
// ========================
func TestService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	catRepo := catMock.NewMockRepository(ctrl)
	service := NewService(repo, catRepo)

	ctx := context.Background()
	catID := uuid.New()

	req := CreateProductRequest{
		CategoryID: catID.String(),
		Name:       "iPhone 15",
		Price:      15000000,
		Stock:      10,
	}

	t.Run("Success", func(t *testing.T) {
		repoID := uuid.New()

		catRepo.EXPECT().
			GetByID(ctx, catID).
			Return(dbgen.Category{ID: catID}, nil)

		repo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(dbgen.Product{ID: repoID}, nil)

		repo.EXPECT().
			GetByID(ctx, repoID).
			Return(dbgen.GetProductByIDRow{
				ID:           repoID,
				Name:         req.Name,
				Price:        "15000000.00",
				Stock:        10,
				IsActive:     dbgen.NewNullBool(true),
				CategoryName: "Phone",
				CreatedAt:    time.Now(),
			}, nil)

		res, err := service.Create(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, req.Name, res.Name)
	})

	t.Run("Invalid Category ID", func(t *testing.T) {
		_, err := service.Create(ctx, CreateProductRequest{
			CategoryID: "invalid-uuid",
		})

		assert.Error(t, err)
		assert.Equal(t, "invalid category id", err.Error())
	})

	t.Run("Category Not Found", func(t *testing.T) {
		catRepo.EXPECT().
			GetByID(ctx, catID).
			Return(dbgen.Category{}, errors.New("not found"))

		_, err := service.Create(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "category not found", err.Error())
	})
}

// ========================
// GET BY ID (ADMIN)
// ========================
func TestService_GetByIDAdmin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	service := NewService(repo, nil)

	ctx := context.Background()
	id := uuid.New()

	t.Run("Success", func(t *testing.T) {
		repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.GetProductByIDRow{
				ID:       id,
				Name:     "Macbook",
				Price:    "20000000.00",
				Stock:    5,
				IsActive: dbgen.NewNullBool(true),
			}, nil)

		res, err := service.GetByIDAdmin(ctx, id.String())

		assert.NoError(t, err)
		assert.Equal(t, "Macbook", res.Name)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		_, err := service.GetByIDAdmin(ctx, "invalid-id")

		assert.Error(t, err)
		assert.Equal(t, "invalid product id", err.Error())
	})

	t.Run("Not Found", func(t *testing.T) {
		repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.GetProductByIDRow{}, errors.New("not found"))

		_, err := service.GetByIDAdmin(ctx, id.String())
		assert.Error(t, err)
	})
}

// ========================
// UPDATE PRODUCT
// ========================
func TestService_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	service := NewService(repo, nil)

	ctx := context.Background()
	id := uuid.New()

	existing := dbgen.GetProductByIDRow{
		ID:    id,
		Name:  "Old Name",
		Price: "100.00",
		Stock: 5,
	}

	req := UpdateProductRequest{
		Name:  "New Name",
		Price: 200,
	}

	t.Run("Success", func(t *testing.T) {
		repo.EXPECT().
			GetByID(ctx, id).
			Return(existing, nil)

		repo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(dbgen.Product{}, nil)

		repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.GetProductByIDRow{
				ID:    id,
				Name:  req.Name,
				Price: "200.00",
				Stock: 5,
			}, nil)

		res, err := service.Update(ctx, id.String(), req)

		assert.NoError(t, err)
		assert.Equal(t, req.Name, res.Name)
	})

	t.Run("Product Not Found", func(t *testing.T) {
		repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.GetProductByIDRow{}, errors.New("not found"))

		_, err := service.Update(ctx, id.String(), req)

		assert.Error(t, err)
		assert.Equal(t, "product not found", err.Error())
	})
}

// ========================
// DELETE PRODUCT
// ========================
func TestService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	service := NewService(repo, nil)

	ctx := context.Background()
	id := uuid.New()

	t.Run("Success", func(t *testing.T) {
		repo.EXPECT().
			Delete(ctx, id).
			Return(nil)

		err := service.Delete(ctx, id.String())
		assert.NoError(t, err)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		err := service.Delete(ctx, "invalid-id")
		assert.Error(t, err)
	})
}

// ========================
// RESTORE PRODUCT
// ========================
func TestService_Restore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	service := NewService(repo, nil)

	ctx := context.Background()
	id := uuid.New()

	repo.EXPECT().
		Restore(ctx, id).
		Return(dbgen.Product{}, nil)

	repo.EXPECT().
		GetByID(ctx, id).
		Return(dbgen.GetProductByIDRow{
			ID:    id,
			Name:  "Restored",
			Price: "100.00",
		}, nil)

	res, err := service.Restore(ctx, id.String())

	assert.NoError(t, err)
	assert.Equal(t, "Restored", res.Name)
}
