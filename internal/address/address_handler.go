package address

import (
	"go-gadget-api/internal/pkg/apperror"
	"go-gadget-api/internal/pkg/response"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

// GET /addresses
func (h *Handler) List(c *gin.Context) {
	userID := c.GetString("user_id")

	res, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FAILED", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// GET /addresses/:id
func (h *Handler) Detail(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	res, err := h.service.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		// Log error secara internal untuk debugging
		log.Printf("GetByID error: %v", err)
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "address not found or invalid id", nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// POST /addresses
func (h *Handler) Create(c *gin.Context) {
	userID := c.GetString("user_id")

	var req CreateAddressRequest
	req.UserID = userID
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
		return
	}

	res, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FAILED", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusCreated, res, nil)
}

// PUT /addresses/:id
func (h *Handler) Update(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	var req UpdateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 1. Map dulu ke AppError
		mappedErr := apperror.MapValidationError(err)

		// 2. Cast ke *apperror.AppError untuk akses field Message
		if ae, ok := mappedErr.(*apperror.AppError); ok {
			response.Error(c, ae.HTTPStatus, ae.Code, ae.Message, nil)
			return
		}

		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
		return
	}

	res, err := h.service.Update(c.Request.Context(), id, userID, req)
	if err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// DELETE /addresses/:id
func (h *Handler) Delete(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	if err := h.service.Delete(c.Request.Context(), id, userID); err != nil {
		response.Error(c, http.StatusInternalServerError, "FAILED", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Address deleted"}, nil)
}

// GET /admin/addresses
func (h *Handler) ListAdmin(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	data, total, err := h.service.ListAdmin(
		c.Request.Context(),
		page,
		limit,
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FAILED", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"data": data,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	}, nil)
}
