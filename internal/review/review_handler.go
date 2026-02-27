package review

import (
	"go-gadget-api/internal/pkg/apperror"
	"go-gadget-api/internal/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) Create(c *gin.Context) {
	userID, _ := c.Get("user_id")
	productSlug := c.Param("slug")

	var req CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperror.Wrap(err, apperror.CodeInvalidInput, "Invalid request body", http.StatusBadRequest)
		httpErr := apperror.ToHTTP(appErr)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, err.Error())
		return
	}

	// Parsing userID ke string dilakukan langsung, validasi eksistensi ada di service/middleware
	uid, _ := userID.(string)

	res, err := h.service.Create(c.Request.Context(), uid, productSlug, req)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusCreated, res, nil)
}

func (h *Handler) GetReviewsByProductSlug(c *gin.Context) {
	productSlug := c.Param("slug")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	res, err := h.service.GetByProductSlug(c.Request.Context(), productSlug, page, limit)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	meta := response.NewPaginationMeta(res.Total, page, limit)

	response.Success(c, http.StatusOK, gin.H{
		"items": res,
		"meta":  meta,
	}, nil)
}

func (h *Handler) GetReviewsByUserID(c *gin.Context) {
	userID := c.GetString("user_id")

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	// 3. Business logic validation di service
	res, err := h.service.GetByUserID(c.Request.Context(), userID, page, limit)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	meta := response.NewPaginationMeta(res.Total, page, limit)

	response.Success(c, http.StatusOK, gin.H{
		"items": res,
		"meta":  meta,
	}, nil)
}

func (h *Handler) UpdateReview(c *gin.Context) {
	userID, _ := c.Get("user_id")
	reviewID := c.Param("id")

	var req UpdateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperror.Wrap(err, apperror.CodeInvalidInput, "Invalid request body", http.StatusBadRequest)
		httpErr := apperror.ToHTTP(appErr)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, err.Error())
		return
	}

	uid, _ := userID.(string)
	res, err := h.service.Update(c.Request.Context(), reviewID, uid, req)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

func (h *Handler) DeleteReview(c *gin.Context) {
	userID, _ := c.Get("user_id")
	reviewID := c.Param("id")

	uid, _ := userID.(string)
	err := h.service.Delete(c.Request.Context(), reviewID, uid)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Review deleted successfully",
	}, nil)
}
