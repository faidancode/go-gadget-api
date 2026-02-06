package address

import (
	"go-gadget-api/internal/pkg/response"
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
func (ctrl *Handler) List(c *gin.Context) {
	userID := c.GetString("user_id")

	res, err := ctrl.service.List(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FAILED", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// POST /addresses
func (ctrl *Handler) Create(c *gin.Context) {
	userID := c.GetString("user_id")

	var req CreateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
		return
	}
	req.UserID = userID

	res, err := ctrl.service.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FAILED", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusCreated, res, nil)
}

// PUT /addresses/:id
func (ctrl *Handler) Update(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	var req UpdateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
		return
	}

	res, err := ctrl.service.Update(c.Request.Context(), id, userID, req)
	if err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// DELETE /addresses/:id
func (ctrl *Handler) Delete(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	if err := ctrl.service.Delete(c.Request.Context(), id, userID); err != nil {
		response.Error(c, http.StatusInternalServerError, "FAILED", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Address deleted"}, nil)
}

// GET /admin/addresses
func (ctrl *Handler) ListAdmin(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	data, total, err := ctrl.service.ListAdmin(
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
