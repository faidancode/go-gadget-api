package review

import (
	"context"
	"database/sql"
	"gadget-api/internal/dbgen"

	"github.com/google/uuid"
)

//go:generate mockgen -source=review_repo.go -destination=../mock/review/review_repo_mock.go -package=mock
type Repository interface {
	WithTx(tx dbgen.DBTX) Repository
	Create(ctx context.Context, arg dbgen.CreateReviewParams) (dbgen.Review, error)
	GetByID(ctx context.Context, id uuid.UUID) (dbgen.GetReviewByIDRow, error)
	GetByProductID(ctx context.Context, productID uuid.UUID, limit, offset int32) ([]dbgen.GetReviewsByProductIDRow, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]dbgen.GetReviewsByUserIDRow, error)
	CountByProductID(ctx context.Context, productID uuid.UUID) (int64, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	GetAverageRating(ctx context.Context, productID uuid.UUID) (float64, error)
	CheckExists(ctx context.Context, userID, productID uuid.UUID) (bool, error)
	CheckUserPurchased(ctx context.Context, userID, productID uuid.UUID) (bool, error)
	GetCompletedOrder(ctx context.Context, userID, productID uuid.UUID) (uuid.UUID, error)
	Update(ctx context.Context, arg dbgen.UpdateReviewParams) (dbgen.Review, error)
	Delete(ctx context.Context, id uuid.UUID) error
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

func (r *repository) Create(ctx context.Context, arg dbgen.CreateReviewParams) (dbgen.Review, error) {
	return r.queries.CreateReview(ctx, arg)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (dbgen.GetReviewByIDRow, error) {
	return r.queries.GetReviewByID(ctx, id)
}

func (r *repository) GetByProductID(ctx context.Context, productID uuid.UUID, limit, offset int32) ([]dbgen.GetReviewsByProductIDRow, error) {
	return r.queries.GetReviewsByProductID(ctx, dbgen.GetReviewsByProductIDParams{
		ProductID: productID,
		Limit:     limit,
		Offset:    offset,
	})
}

func (r *repository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]dbgen.GetReviewsByUserIDRow, error) {
	return r.queries.GetReviewsByUserID(ctx, dbgen.GetReviewsByUserIDParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
}

func (r *repository) CountByProductID(ctx context.Context, productID uuid.UUID) (int64, error) {
	return r.queries.CountReviewsByProductID(ctx, productID)
}

func (r *repository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	return r.queries.CountReviewsByUserID(ctx, userID)
}

func (r *repository) GetAverageRating(ctx context.Context, productID uuid.UUID) (float64, error) {
	result, err := r.queries.GetAverageRatingByProductID(ctx, productID)
	if err != nil {
		return 0, err
	}
	return result.(float64), nil
}

func (r *repository) CheckExists(ctx context.Context, userID, productID uuid.UUID) (bool, error) {
	return r.queries.CheckReviewExists(ctx, dbgen.CheckReviewExistsParams{
		UserID:    userID,
		ProductID: productID,
	})
}

func (r *repository) CheckUserPurchased(ctx context.Context, userID, productID uuid.UUID) (bool, error) {
	return r.queries.CheckUserPurchasedProduct(ctx, dbgen.CheckUserPurchasedProductParams{
		UserID:    userID,
		ProductID: productID,
	})
}

func (r *repository) GetCompletedOrder(ctx context.Context, userID, productID uuid.UUID) (uuid.UUID, error) {
	return r.queries.GetCompletedOrderForReview(ctx, dbgen.GetCompletedOrderForReviewParams{
		UserID:    userID,
		ProductID: productID,
	})
}

func (r *repository) Update(ctx context.Context, arg dbgen.UpdateReviewParams) (dbgen.Review, error) {
	return r.queries.UpdateReview(ctx, arg)
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteReview(ctx, id)
}
