package review_test

import (
	"context"
	"database/sql"
	"testing"

	"gadget-api/internal/dbgen"
	"gadget-api/internal/review"

	productMock "gadget-api/internal/mock/product"
	reviewMock "gadget-api/internal/mock/review"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ======================= HELPERS =======================

type reviewDeps struct {
	db          *sql.DB
	sqlMock     sqlmock.Sqlmock
	service     review.Service
	repo        *reviewMock.MockRepository
	productRepo *productMock.MockRepository
}

func setupReviewTest(t *testing.T) *reviewDeps {
	t.Helper()
	ctrl := gomock.NewController(t)
	db, sqlMock, _ := sqlmock.New()

	repo := reviewMock.NewMockRepository(ctrl)
	productRepo := productMock.NewMockRepository(ctrl)

	svc := review.NewService(db, repo, productRepo)

	return &reviewDeps{
		db:          db,
		sqlMock:     sqlMock,
		service:     svc,
		repo:        repo,
		productRepo: productRepo,
	}
}

// Helper untuk ekspektasi transaksi
func expectTx(sqlMock sqlmock.Sqlmock, shouldCommit bool) {
	if shouldCommit {
		sqlMock.ExpectBegin()
		sqlMock.ExpectCommit()
	} else {
		sqlMock.ExpectBegin()
		sqlMock.ExpectRollback()
	}
}

// ======================= CREATE =======================

func TestReviewService_Create(t *testing.T) {
	deps := setupReviewTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	userID := uuid.New()
	productID := uuid.New()
	orderID := uuid.New()
	productSlug := "gadget-xyz"

	req := review.CreateReviewRequest{Rating: 5, Comment: "Mantap!"}

	t.Run("positive - success create review", func(t *testing.T) {
		expectTx(deps.sqlMock, true)

		deps.productRepo.EXPECT().GetBySlug(ctx, productSlug).Return(dbgen.GetProductBySlugRow{ID: productID}, nil)
		deps.repo.EXPECT().CheckExists(ctx, userID, productID).Return(false, nil)
		deps.repo.EXPECT().CheckUserPurchased(ctx, userID, productID).Return(true, nil)
		deps.repo.EXPECT().GetCompletedOrder(ctx, userID, productID).Return(orderID, nil)

		deps.repo.EXPECT().WithTx(gomock.Any()).Return(deps.repo)
		deps.repo.EXPECT().Create(ctx, gomock.Any()).Return(dbgen.Review{ID: uuid.New()}, nil)

		// MapToReviewResponse calls GetByID
		deps.repo.EXPECT().GetByID(ctx, gomock.Any()).Return(dbgen.GetReviewByIDRow{
			ID: uuid.New(), UserID: userID, UserName: "John", Comment: req.Comment,
		}, nil)

		res, err := deps.service.Create(ctx, userID.String(), productSlug, req)
		assert.NoError(t, err)
		assert.Equal(t, req.Comment, res.Comment)
	})

	t.Run("negative - already reviewed", func(t *testing.T) {
		deps.productRepo.EXPECT().GetBySlug(ctx, productSlug).Return(dbgen.GetProductBySlugRow{ID: productID}, nil)
		deps.repo.EXPECT().CheckExists(ctx, userID, productID).Return(true, nil)

		_, err := deps.service.Create(ctx, userID.String(), productSlug, req)
		assert.Error(t, err)
		assert.Equal(t, review.ErrReviewAlreadyExists, err)
	})
}

// ======================= ELIGIBILITY =======================

func TestReviewService_CheckEligibility(t *testing.T) {
	deps := setupReviewTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	userID := uuid.New()
	productSlug := "gadget-xyz"
	productID := uuid.New()

	t.Run("positive - user is eligible", func(t *testing.T) {
		deps.productRepo.EXPECT().GetBySlug(ctx, productSlug).Return(dbgen.GetProductBySlugRow{ID: productID}, nil)
		deps.repo.EXPECT().CheckExists(ctx, userID, productID).Return(false, nil)
		deps.repo.EXPECT().CheckUserPurchased(ctx, userID, productID).Return(true, nil)

		res, err := deps.service.CheckEligibility(ctx, userID.String(), productSlug)
		assert.NoError(t, err)
		assert.True(t, res.CanReview)
	})

	t.Run("negative - user not purchased", func(t *testing.T) {
		deps.productRepo.EXPECT().GetBySlug(ctx, productSlug).Return(dbgen.GetProductBySlugRow{ID: productID}, nil)
		deps.repo.EXPECT().CheckExists(ctx, userID, productID).Return(false, nil)
		deps.repo.EXPECT().CheckUserPurchased(ctx, userID, productID).Return(false, nil)

		res, err := deps.service.CheckEligibility(ctx, userID.String(), productSlug)
		assert.NoError(t, err)
		assert.False(t, res.CanReview)
		assert.Equal(t, "You must purchase this product before reviewing", res.Reason)
	})
}

// ======================= UPDATE =======================

func TestReviewService_Update(t *testing.T) {
	deps := setupReviewTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	reviewID := uuid.New()
	userID := uuid.New()
	req := review.UpdateReviewRequest{Rating: 4, Comment: "Update comment"}

	t.Run("positive - success update", func(t *testing.T) {
		expectTx(deps.sqlMock, true)

		deps.repo.EXPECT().GetByID(ctx, reviewID).Return(dbgen.GetReviewByIDRow{ID: reviewID, UserID: userID}, nil)
		deps.repo.EXPECT().WithTx(gomock.Any()).Return(deps.repo)
		deps.repo.EXPECT().Update(ctx, gomock.Any()).Return(dbgen.Review{}, nil)
		deps.repo.EXPECT().GetByID(ctx, reviewID).Return(dbgen.GetReviewByIDRow{ID: reviewID, Comment: req.Comment}, nil)

		res, err := deps.service.Update(ctx, reviewID.String(), userID.String(), req)
		assert.NoError(t, err)
		assert.Equal(t, req.Comment, res.Comment)
	})

	t.Run("negative - unauthorized update", func(t *testing.T) {
		otherUser := uuid.New()
		deps.repo.EXPECT().GetByID(ctx, reviewID).Return(dbgen.GetReviewByIDRow{ID: reviewID, UserID: otherUser}, nil)

		_, err := deps.service.Update(ctx, reviewID.String(), userID.String(), req)
		assert.Error(t, err)
		assert.Equal(t, review.ErrUnauthorizedReview, err)
	})
}

// ======================= DELETE =======================

func TestReviewService_Delete(t *testing.T) {
	deps := setupReviewTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	reviewID := uuid.New()
	userID := uuid.New()

	t.Run("positive - success delete", func(t *testing.T) {
		expectTx(deps.sqlMock, true)

		deps.repo.EXPECT().GetByID(ctx, reviewID).Return(dbgen.GetReviewByIDRow{ID: reviewID, UserID: userID}, nil)
		deps.repo.EXPECT().WithTx(gomock.Any()).Return(deps.repo)
		deps.repo.EXPECT().Delete(ctx, reviewID).Return(nil)

		err := deps.service.Delete(ctx, reviewID.String(), userID.String())
		assert.NoError(t, err)
	})

	t.Run("negative - review not found", func(t *testing.T) {
		deps.repo.EXPECT().GetByID(ctx, reviewID).Return(dbgen.GetReviewByIDRow{}, sql.ErrNoRows)

		err := deps.service.Delete(ctx, reviewID.String(), userID.String())
		assert.Error(t, err)
		assert.Equal(t, review.ErrReviewNotFound, err)
	})
}

// ======================= LISTS =======================

func TestReviewService_GetByProductSlug(t *testing.T) {
	deps := setupReviewTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	slug := "iphone"
	pid := uuid.New()

	t.Run("positive - list success", func(t *testing.T) {
		deps.productRepo.EXPECT().GetBySlug(ctx, slug).Return(dbgen.GetProductBySlugRow{ID: pid}, nil)
		deps.repo.EXPECT().GetByProductID(ctx, pid, int32(10), int32(0)).Return([]dbgen.GetReviewsByProductIDRow{{ID: uuid.New()}}, nil)
		deps.repo.EXPECT().CountByProductID(ctx, pid).Return(int64(1), nil)

		res, err := deps.service.GetByProductSlug(ctx, slug, 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
	})

	t.Run("negative - invalid page", func(t *testing.T) {
		// Service handle page < 1 as 1
		deps.productRepo.EXPECT().GetBySlug(ctx, slug).Return(dbgen.GetProductBySlugRow{ID: pid}, nil)
		deps.repo.EXPECT().GetByProductID(ctx, pid, int32(10), int32(0)).Return(nil, nil)
		deps.repo.EXPECT().CountByProductID(ctx, pid).Return(int64(0), nil)

		res, err := deps.service.GetByProductSlug(ctx, slug, 0, 10)
		assert.NoError(t, err)
		assert.Equal(t, 1, res.Page)
	})
}
