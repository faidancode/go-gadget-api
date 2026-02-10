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
	UserID             string    `json:"userdId"`
	UserName           string    `json:"userdName"`
	ProductID          string    `json:"productdId"`
	Rating             int32     `json:"rating"`
	Comment            string    `json:"comment"`
	IsVerifiedPurchase bool      `json:"isVerifiedPurchase"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type ReviewSummaryResponse struct {
	ID        string    `json:"id"`
	UserName  string    `json:"userdName"`
	Rating    int32     `json:"rating"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"createdAt"`
}

type UserReviewResponse struct {
	ID          string    `json:"id"`
	ProductID   string    `json:"productdId"`
	ProductName string    `json:"productdName"`
	ProductSlug string    `json:"productSlug"`
	Rating      int32     `json:"rating"`
	Comment     string    `json:"comment"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ReviewListResponse struct {
	Reviews []ReviewResponse `json:"reviews"`
	Total   int64            `json:"total"`
	Page    int              `json:"page"`
	Limit   int              `json:"limit"`
}

type UserReviewListResponse struct {
	Reviews       []UserReviewResponse `json:"reviews"`
	Total         int64                `json:"total"`
	Page          int                  `json:"page"`
	Limit         int                  `json:"limit"`
	AverageRating float64              `json:"averageRating"`
	TotalReviews  int64                `json:"totalReviews"`
	RatingCounts  map[int]int64        `json:"ratingCounts"`
}

type ReviewEligibilityResponse struct {
	CanReview       bool   `json:"canReview"`
	Reason          string `json:"reason,omitempty"`
	HasPurchased    bool   `json:"hasPurchased"`
	AlreadyReviewed bool   `json:"alreadyReviewed"`
}

type ReviewStatsResponse struct {
	AverageRating   float64       `json:"averageRating"`
	RatingCount     int64         `json:"ratingCount"`
	RatingBreakdown map[int]int64 `json:"ratingBreakdown,omitempty"`
}
