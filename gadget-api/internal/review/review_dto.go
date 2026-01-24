package review

import "time"

// ==================== REQUEST STRUCTS ====================

type CreateReviewRequest struct {
	Rating  int32  `json:"rating" validate:"required,min=1,max=5"`
	Comment string `json:"comment" validate:"required,min=10,max=1000"`
}

type UpdateReviewRequest struct {
	Rating  int32  `json:"rating" validate:"required,min=1,max=5"`
	Comment string `json:"comment" validate:"required,min=10,max=1000"`
}

type GetReviewsRequest struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

// ==================== RESPONSE STRUCTS ====================

type ReviewResponse struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	UserName           string    `json:"user_name"`
	ProductID          string    `json:"product_id"`
	Rating             int32     `json:"rating"`
	Comment            string    `json:"comment"`
	IsVerifiedPurchase bool      `json:"is_verified_purchase"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ReviewSummaryResponse struct {
	ID        string    `json:"id"`
	UserName  string    `json:"user_name"`
	Rating    int32     `json:"rating"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

type UserReviewResponse struct {
	ID          string    `json:"id"`
	ProductID   string    `json:"product_id"`
	ProductName string    `json:"product_name"`
	ProductSlug string    `json:"product_slug"`
	Rating      int32     `json:"rating"`
	Comment     string    `json:"comment"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ReviewListResponse struct {
	Reviews []ReviewResponse `json:"reviews"`
	Total   int64            `json:"total"`
	Page    int              `json:"page"`
	Limit   int              `json:"limit"`
}

type UserReviewListResponse struct {
	Reviews []UserReviewResponse `json:"reviews"`
	Total   int64                `json:"total"`
	Page    int                  `json:"page"`
	Limit   int                  `json:"limit"`
}

type ReviewEligibilityResponse struct {
	CanReview       bool   `json:"can_review"`
	Reason          string `json:"reason,omitempty"`
	HasPurchased    bool   `json:"has_purchased"`
	AlreadyReviewed bool   `json:"already_reviewed"`
}

type ReviewStatsResponse struct {
	AverageRating   float64       `json:"average_rating"`
	TotalReviews    int64         `json:"total_reviews"`
	RatingBreakdown map[int]int64 `json:"rating_breakdown,omitempty"` // optional for future
}
