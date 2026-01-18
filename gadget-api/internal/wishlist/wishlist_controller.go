package wishlist

import (
	"gadget-api/internal/pkg/apperror"
	"gadget-api/internal/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	service Service
}

func NewController(svc Service) *Controller {
	return &Controller{service: svc}
}

// Create adds a product to user's wishlist
// POST /wishlist
func (ctrl *Controller) Create(c *gin.Context) {
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

	res, err := ctrl.service.Create(c.Request.Context(), userID.(string), req.ProductID)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusCreated, res, nil)
}

// List retrieves all items in user's wishlist
// GET /wishlist
func (ctrl *Controller) List(c *gin.Context) {
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

	res, err := ctrl.service.List(c.Request.Context(), userID.(string))
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// Delete removes a product from user's wishlist
// DELETE /wishlist
func (ctrl *Controller) Delete(c *gin.Context) {
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

	var req DeleteItemRequest
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

	err := ctrl.service.Delete(c.Request.Context(), userID.(string), req.ProductID)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Product removed from wishlist successfully",
	}, nil)
}
