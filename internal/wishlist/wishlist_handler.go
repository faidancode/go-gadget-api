package wishlist

import (
	"go-gadget-api/internal/pkg/apperror"
	"go-gadget-api/internal/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{service: svc}
}

// POST /wishlist
func (ctrl *Handler) Create(c *gin.Context) {
	userID := c.GetString("user_id_validated")
	if userID == "" {
		response.Error(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		)
		return
	}

	var req AddItemRequest
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

	res, err := ctrl.service.Create(c.Request.Context(), userID, req.ProductID)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusCreated, res, nil)
}

// GET /wishlist
func (ctrl *Handler) List(c *gin.Context) {
	userID := c.GetString("user_id_validated")
	if userID == "" {
		response.Error(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		)
		return
	}

	res, err := ctrl.service.List(c.Request.Context(), userID)
	if err != nil {
		httpErr := apperror.ToHTTP(err)

		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// DELETE /wishlist
func (ctrl *Handler) Delete(c *gin.Context) {
	userID := c.GetString("user_id_validated")
	if userID == "" {
		response.Error(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		)
		return
	}

	productID := c.Param("productId")
	if productID == "" {
		response.Error(
			c,
			http.StatusBadRequest,
			"INVALID_INPUT",
			"productId is required",
			nil,
		)
		return
	}

	err := ctrl.service.Delete(c.Request.Context(), userID, productID)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Product removed from wishlist successfully",
	}, nil)
}
