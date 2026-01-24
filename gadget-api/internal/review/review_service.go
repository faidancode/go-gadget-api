package review

import (
	"context"
	"database/sql"
	"gadget-api/internal/dbgen"
	"gadget-api/internal/pkg/apperror"
	"gadget-api/internal/product"
	reviewerrors "gadget-api/internal/review/errors"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Product struct minimal yang dibutuhkan
type Product struct {
	ID   uuid.UUID
	Slug string
	Name string
}

//go:generate mockgen -source=review_service.go -destination=../mock/review/review_service_mock.go -package=mock
type Service interface {
	Create(ctx context.Context, userID, productSlug string, req CreateReviewRequest) (ReviewResponse, error)
	GetByProductSlug(ctx context.Context, productSlug string, page, limit int) (ReviewListResponse, error)
	GetByUserID(ctx context.Context, userID string, page, limit int) (UserReviewListResponse, error)
	CheckEligibility(ctx context.Context, userID, productSlug string) (ReviewEligibilityResponse, error)
	Update(ctx context.Context, reviewID, userID string, req UpdateReviewRequest) (ReviewResponse, error)
	Delete(ctx context.Context, reviewID, userID string) error
}

type service struct {
	repo        Repository
	productRepo product.Repository
	db          *sql.DB
	validate    *validator.Validate
}

func NewService(db *sql.DB, r Repository, pr product.Repository) Service {
	return &service{
		db:          db,
		repo:        r,
		productRepo: pr,
		validate:    validator.New(),
	}
}

// Create creates a new review for a product
func (s *service) Create(ctx context.Context, userID, productSlug string, req CreateReviewRequest) (ReviewResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return ReviewResponse{}, apperror.MapValidationError(err)
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return ReviewResponse{}, reviewerrors.ErrInvalidReviewID
	}

	// 1. Get product by slug
	product, err := s.productRepo.GetBySlug(ctx, productSlug)
	if err != nil {
		if err == sql.ErrNoRows {
			return ReviewResponse{}, reviewerrors.ErrProductNotFound
		}
		return ReviewResponse{}, reviewerrors.ErrReviewFailed
	}

	// 2. Check if user already reviewed this product
	exists, err := s.repo.CheckExists(ctx, uid, product.ID)
	if err != nil {
		return ReviewResponse{}, reviewerrors.ErrReviewFailed
	}
	if exists {
		return ReviewResponse{}, reviewerrors.ErrReviewAlreadyExists
	}

	// 3. Check if user purchased this product
	purchased, err := s.repo.CheckUserPurchased(ctx, uid, product.ID)
	if err != nil {
		return ReviewResponse{}, reviewerrors.ErrReviewFailed
	}
	if !purchased {
		return ReviewResponse{}, reviewerrors.ErrNotPurchased
	}

	// 4. Get completed order for this product
	orderID, err := s.repo.GetCompletedOrder(ctx, uid, product.ID)
	if err != nil {
		return ReviewResponse{}, reviewerrors.ErrOrderNotCompleted
	}

	// 5. Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReviewResponse{}, reviewerrors.ErrReviewFailed
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)

	// 6. Create review
	review, err := qtx.Create(ctx, CreateReviewParams{
		UserID:             uid,
		ProductID:          product.ID,
		OrderID:            orderID,
		Rating:             req.Rating,
		Comment:            req.Comment,
		IsVerifiedPurchase: true,
	})
	if err != nil {
		return ReviewResponse{}, reviewerrors.ErrReviewFailed
	}

	// 7. Commit transaction
	if err := tx.Commit(); err != nil {
		return ReviewResponse{}, reviewerrors.ErrReviewFailed
	}

	// 8. Fetch complete review data with user name
	reviewDetail, err := s.repo.GetByID(ctx, review.ID)
	if err != nil {
		return ReviewResponse{}, reviewerrors.ErrReviewFailed
	}

	return s.mapToReviewResponse(reviewDetail), nil
}

// GetByProductSlug retrieves all reviews for a product
func (s *service) GetByProductSlug(ctx context.Context, productSlug string, page, limit int) (ReviewListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	// 1. Get product by slug
	product, err := s.productRepo.GetBySlug(ctx, productSlug)
	if err != nil {
		if err == sql.ErrNoRows {
			return ReviewListResponse{}, reviewerrors.ErrProductNotFound
		}
		return ReviewListResponse{}, reviewerrors.ErrReviewFailed
	}

	// 2. Get reviews
	offset := int32((page - 1) * limit)
	reviews, err := s.repo.GetByProductID(ctx, product.ID, int32(limit), offset)
	if err != nil {
		return ReviewListResponse{}, reviewerrors.ErrReviewFailed
	}

	// 3. Get total count
	total, err := s.repo.CountByProductID(ctx, product.ID)
	if err != nil {
		return ReviewListResponse{}, reviewerrors.ErrReviewFailed
	}

	// 4. Map to response
	var reviewResponses []ReviewResponse
	for _, r := range reviews {
		reviewResponses = append(reviewResponses, ReviewResponse{
			ID:                 r.ID.String(),
			UserID:             r.UserID.String(),
			UserName:           r.UserName,
			ProductID:          r.ProductID.String(),
			Rating:             r.Rating,
			Comment:            r.Comment,
			IsVerifiedPurchase: r.IsVerifiedPurchase,
			CreatedAt:          r.CreatedAt,
			UpdatedAt:          r.UpdatedAt,
		})
	}

	return ReviewListResponse{
		Reviews: reviewResponses,
		Total:   total,
		Page:    page,
		Limit:   limit,
	}, nil
}

// GetByUserID retrieves all reviews by a user
func (s *service) GetByUserID(ctx context.Context, userID string, page, limit int) (UserReviewListResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return UserReviewListResponse{}, reviewerrors.ErrInvalidReviewID
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	// 1. Get reviews
	offset := int32((page - 1) * limit)
	reviews, err := s.repo.GetByUserID(ctx, uid, int32(limit), offset)
	if err != nil {
		return UserReviewListResponse{}, reviewerrors.ErrReviewFailed
	}

	// 2. Get total count
	total, err := s.repo.CountByUserID(ctx, uid)
	if err != nil {
		return UserReviewListResponse{}, reviewerrors.ErrReviewFailed
	}

	// 3. Map to response
	var reviewResponses []UserReviewResponse
	for _, r := range reviews {
		reviewResponses = append(reviewResponses, UserReviewResponse{
			ID:          r.ID.String(),
			ProductID:   r.ProductID.String(),
			ProductName: r.ProductName,
			ProductSlug: r.ProductSlug,
			Rating:      r.Rating,
			Comment:     r.Comment,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
		})
	}

	return UserReviewListResponse{
		Reviews: reviewResponses,
		Total:   total,
		Page:    page,
		Limit:   limit,
	}, nil
}

// CheckEligibility checks if a user can review a product
func (s *service) CheckEligibility(ctx context.Context, userID, productSlug string) (ReviewEligibilityResponse, error) {
	// If user not authenticated, return not eligible
	if userID == "" {
		return ReviewEligibilityResponse{
			CanReview:       false,
			Reason:          "You must be logged in to review products",
			HasPurchased:    false,
			AlreadyReviewed: false,
		}, nil
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return ReviewEligibilityResponse{}, reviewerrors.ErrInvalidReviewID
	}

	// 1. Get product by slug
	product, err := s.productRepo.GetBySlug(ctx, productSlug)
	if err != nil {
		if err == sql.ErrNoRows {
			return ReviewEligibilityResponse{}, reviewerrors.ErrProductNotFound
		}
		return ReviewEligibilityResponse{}, reviewerrors.ErrReviewFailed
	}

	// 2. Check if already reviewed
	alreadyReviewed, err := s.repo.CheckExists(ctx, uid, product.ID)
	if err != nil {
		return ReviewEligibilityResponse{}, reviewerrors.ErrReviewFailed
	}

	if alreadyReviewed {
		return ReviewEligibilityResponse{
			CanReview:       false,
			Reason:          "You have already reviewed this product",
			HasPurchased:    true, // assume true if they could review
			AlreadyReviewed: true,
		}, nil
	}

	// 3. Check if purchased
	hasPurchased, err := s.repo.CheckUserPurchased(ctx, uid, product.ID)
	if err != nil {
		return ReviewEligibilityResponse{}, reviewerrors.ErrReviewFailed
	}

	if !hasPurchased {
		return ReviewEligibilityResponse{
			CanReview:       false,
			Reason:          "You must purchase this product before reviewing",
			HasPurchased:    false,
			AlreadyReviewed: false,
		}, nil
	}

	// All checks passed
	return ReviewEligibilityResponse{
		CanReview:       true,
		HasPurchased:    true,
		AlreadyReviewed: false,
	}, nil
}

// Update updates an existing review
func (s *service) Update(ctx context.Context, reviewID, userID string, req UpdateReviewRequest) (ReviewResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return ReviewResponse{}, apperror.MapValidationError(err)
	}

	rid, err := uuid.Parse(reviewID)
	if err != nil {
		return ReviewResponse{}, reviewerrors.ErrInvalidReviewID
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return ReviewResponse{}, reviewerrors.ErrInvalidReviewID
	}

	// 1. Get existing review
	review, err := s.repo.GetByID(ctx, rid)
	if err != nil {
		if err == sql.ErrNoRows {
			return ReviewResponse{}, reviewerrors.ErrReviewNotFound
		}
		return ReviewResponse{}, reviewerrors.ErrReviewFailed
	}

	// 2. Check authorization
	if review.UserID != uid {
		return ReviewResponse{}, reviewerrors.ErrUnauthorizedReview
	}

	// 3. Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReviewResponse{}, reviewerrors.ErrReviewFailed
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)

	// 4. Update review
	_, err = qtx.Update(ctx, UpdateReviewParams{
		ID:      rid,
		Rating:  req.Rating,
		Comment: req.Comment,
	})
	if err != nil {
		return ReviewResponse{}, reviewerrors.ErrReviewFailed
	}

	// 5. Commit transaction
	if err := tx.Commit(); err != nil {
		return ReviewResponse{}, reviewerrors.ErrReviewFailed
	}

	// 6. Fetch updated review
	updatedReview, err := s.repo.GetByID(ctx, rid)
	if err != nil {
		return ReviewResponse{}, reviewerrors.ErrReviewFailed
	}

	return s.mapToReviewResponse(updatedReview), nil
}

// Delete soft deletes a review
func (s *service) Delete(ctx context.Context, reviewID, userID string) error {
	rid, err := uuid.Parse(reviewID)
	if err != nil {
		return reviewerrors.ErrInvalidReviewID
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return reviewerrors.ErrInvalidReviewID
	}

	// 1. Get existing review
	review, err := s.repo.GetByID(ctx, rid)
	if err != nil {
		if err == sql.ErrNoRows {
			return reviewerrors.ErrReviewNotFound
		}
		return reviewerrors.ErrReviewFailed
	}

	// 2. Check authorization
	if review.UserID != uid {
		return reviewerrors.ErrUnauthorizedReview
	}

	// 3. Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return reviewerrors.ErrReviewFailed
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)

	// 4. Delete review
	err = qtx.Delete(ctx, rid)
	if err != nil {
		return reviewerrors.ErrReviewFailed
	}

	// 5. Commit transaction
	if err := tx.Commit(); err != nil {
		return reviewerrors.ErrReviewFailed
	}

	return nil
}

// Helper function to map review to response
func (s *service) mapToReviewResponse(r GetReviewByIDRow) ReviewResponse {
	return ReviewResponse{
		ID:                 r.ID.String(),
		UserID:             r.UserID.String(),
		UserName:           r.UserName,
		ProductID:          r.ProductID.String(),
		Rating:             r.Rating,
		Comment:            r.Comment,
		IsVerifiedPurchase: r.IsVerifiedPurchase,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
	}
}

// Note: These type aliases should match your actual dbgen types
type CreateReviewParams = dbgen.CreateReviewParams
type UpdateReviewParams = dbgen.UpdateReviewParams
type GetReviewByIDRow = dbgen.GetReviewByIDRow
