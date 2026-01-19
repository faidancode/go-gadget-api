package review

import (
	"gadget-api/internal/pkg/apperror"
	"gadget-api/internal/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	service Service
}

func NewController(svc Service) *Controller {
	return &Controller{service: svc}
}

// CreateReview creates a new review for a product
// POST /products/:slug/reviews
func (ctrl *Controller) CreateReview(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		)
		return
	}

	productSlug := c.Param("slug")
	if productSlug == "" {
		httpErr := apperror.ToHTTP(ErrInvalidProductSlug)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	var req CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperror.Wrap(
			err,
			apperror.CodeInvalidInput,
			"Invalid request body",
			http.StatusBadRequest,
		)
		httpErr := apperror.ToHTTP(appErr)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, err.Error())
		return
	}

	res, err := ctrl.service.Create(c.Request.Context(), userID.(string), productSlug, req)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusCreated, res, nil)
}

// GetReviewsByProductSlug retrieves all reviews for a product
// GET /products/:slug/reviews
func (ctrl *Controller) GetReviewsByProductSlug(c *gin.Context) {
	productSlug := c.Param("slug")
	if productSlug == "" {
		httpErr := apperror.ToHTTP(ErrInvalidProductSlug)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	res, err := ctrl.service.GetByProductSlug(c.Request.Context(), productSlug, page, limit)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// GetReviewsByUserID retrieves all reviews by a user
// GET /reviews/user/:userId
func (ctrl *Controller) GetReviewsByUserID(c *gin.Context) {
	// Verify authenticated user
	authenticatedUserID, exists := c.Get("user_id")
	if !exists {
		response.Error(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		)
		return
	}

	userID := c.Param("userId")
	if userID == "" {
		httpErr := apperror.ToHTTP(ErrInvalidReviewID)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	// Optional: Check if user can only view their own reviews
	if userID != authenticatedUserID.(string) {
		response.Error(
			c,
			http.StatusForbidden,
			"FORBIDDEN",
			"You can only view your own reviews",
			nil,
		)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	res, err := ctrl.service.GetByUserID(c.Request.Context(), userID, page, limit)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// CheckReviewEligibility checks if a user can review a product
// GET /products/:slug/reviews/eligibility
func (ctrl *Controller) CheckReviewEligibility(c *gin.Context) {
	// This endpoint supports optional authentication
	userID, _ := c.Get("user_id")
	userIDStr := ""
	if userID != nil {
		userIDStr = userID.(string)
	}

	productSlug := c.Param("slug")
	if productSlug == "" {
		httpErr := apperror.ToHTTP(ErrInvalidProductSlug)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	res, err := ctrl.service.CheckEligibility(c.Request.Context(), userIDStr, productSlug)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// UpdateReview updates an existing review
// PUT /reviews/:id
func (ctrl *Controller) UpdateReview(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		)
		return
	}

	reviewID := c.Param("id")
	if reviewID == "" {
		httpErr := apperror.ToHTTP(ErrInvalidReviewID)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	var req UpdateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperror.Wrap(
			err,
			apperror.CodeInvalidInput,
			"Invalid request body",
			http.StatusBadRequest,
		)
		httpErr := apperror.ToHTTP(appErr)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, err.Error())
		return
	}

	res, err := ctrl.service.Update(c.Request.Context(), reviewID, userID.(string), req)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// DeleteReview soft deletes a review
// DELETE /reviews/:id
func (ctrl *Controller) DeleteReview(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		)
		return
	}

	reviewID := c.Param("id")
	if reviewID == "" {
		httpErr := apperror.ToHTTP(ErrInvalidReviewID)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	err := ctrl.service.Delete(c.Request.Context(), reviewID, userID.(string))
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Review deleted successfully",
	}, nil)
}
