package adapters

import (
	"context"

	"gadget-api/internal/product"
	"gadget-api/internal/review"
)

type ReviewEligibilityAdapter struct {
	reviewSvc review.Service
}

func NewReviewEligibilityAdapter(
	reviewSvc review.Service,
) product.ReviewService {
	return &ReviewEligibilityAdapter{
		reviewSvc: reviewSvc,
	}
}

func (a *ReviewEligibilityAdapter) CheckEligibility(
	ctx context.Context,
	userID string,
	productSlug string,
) (product.EligibilityResponse, error) {

	res, err := a.reviewSvc.CheckEligibility(ctx, userID, productSlug)
	if err != nil {
		return product.EligibilityResponse{}, err
	}

	return product.EligibilityResponse{
		CanReview: res.CanReview,
		Reason:    res.Reason,
	}, nil
}
