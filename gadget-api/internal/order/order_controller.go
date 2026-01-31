package order

import (
	"gadget-api/internal/pkg/apperror"
	"gadget-api/internal/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Controller struct {
	service Service
}

func NewController(svc Service) *Controller {
	return &Controller{service: svc}
}

// ==================== CUSTOMER ENDPOINTS ====================

// Checkout creates a new order from user's cart
// POST /orders
func (ctrl *Controller) Checkout(c *gin.Context) {
	var req CheckoutRequest

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
	req.UserID = userID.(string)

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

	res, err := ctrl.service.Checkout(c.Request.Context(), req)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusCreated, res, nil)
}

// List retrieves all orders for the authenticated user
// GET /orders?page=1&limit=10
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

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	orders, total, err := ctrl.service.List(
		c.Request.Context(),
		userID.(string),
		page,
		limit,
	)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"orders": orders,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	}, nil)
}

// Detail retrieves a single order by ID
// GET /orders/:id
func (ctrl *Controller) Detail(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		httpErr := apperror.ToHTTP(ErrInvalidOrderID)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	res, err := ctrl.service.Detail(c.Request.Context(), orderID)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// Cancel cancels an order (only for PENDING status)
// PATCH /orders/:id/cancel
func (ctrl *Controller) Cancel(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		httpErr := apperror.ToHTTP(ErrInvalidOrderID)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	if err := ctrl.service.Cancel(c.Request.Context(), orderID); err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Order cancelled successfully",
	}, nil)
}

// ==================== ADMIN ENDPOINTS ====================

// ListAdmin retrieves all orders with filters (admin only)
// GET /admin/orders
func (ctrl *Controller) ListAdmin(c *gin.Context) {
	status := c.Query("status")
	search := c.Query("search")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	orders, total, err := ctrl.service.ListAdmin(
		c.Request.Context(),
		status,
		search,
		page,
		limit,
	)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"orders": orders,
		"pagination": gin.H{
			"page":   page,
			"limit":  limit,
			"total":  total,
			"status": status,
			"search": search,
		},
	}, nil)
}

// UpdateStatus updates order status (admin only)
// PATCH /admin/orders/:id/status
func (c *Controller) UpdateStatusByAdmin(ctx *gin.Context) {
	id := ctx.Param("id")

	var req UpdateStatusAdminRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := c.service.UpdateStatusByAdmin(
		ctx.Request.Context(),
		id,
		req.Status,
		req.ReceiptNo,
	)
	if err != nil {
		switch err {
		case ErrOrderNotFound:
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case ErrInvalidOrderID,
			ErrInvalidStatusTransition,
			ErrReceiptRequired:
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	ctx.JSON(http.StatusOK, res)
}

// PATCH /api/v1/orders/:id/complete
func (c *Controller) UpdateStatusByCustomer(ctx *gin.Context) {
	id := ctx.Param("id")

	// Ambil UserID dari middleware Auth
	userIDStr, _ := ctx.Get("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))

	// Langsung paksa status ke COMPLETED karena ini endpoint khusus customer
	res, err := c.service.UpdateStatusByCustomer(ctx.Request.Context(), id, userID, "COMPLETED")
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(200, res)
}
