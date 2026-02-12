package order

import (
	"encoding/json"
	"fmt"
	"go-gadget-api/internal/pkg/apperror"
	"go-gadget-api/internal/pkg/response"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	service Service
	rdb     *redis.Client
}

func NewHandler(svc Service, rdb *redis.Client) *Handler {
	return &Handler{service: svc, rdb: rdb}
}

// ==================== CUSTOMER ENDPOINTS ====================

// Checkout creates a new order from user's cart
// POST /orders
func (ctrl *Handler) Checkout(c *gin.Context) {
	userID := c.GetString("user_id_validated")
	fmt.Printf("[CHECKOUT HANDLER] userID: '%s'\n", userID) // ← Debug

	if userID == "" {
		fmt.Println("[CHECKOUT HANDLER] ERROR: userID is empty!") // ← Debug
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", nil)
		return
	}

	lockKey, _ := c.Get("idempotency_lock_key")

	// Pastikan lock dihapus di akhir request
	defer func() {
		if lockKey != nil {
			ctrl.rdb.Del(c.Request.Context(), lockKey.(string))
		}
	}()

	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("[CHECKOUT HANDLER] Bind error: %v\n", err) // ← Debug
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

	fmt.Printf("[CHECKOUT HANDLER] Calling service.Checkout with userID: '%s'\n", userID) // ← Debug
	res, err := ctrl.service.Checkout(c.Request.Context(), userID, req)
	if err != nil {
		fmt.Printf("[CHECKOUT HANDLER] Service error: %v\n", err) // ← Debug
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	if cacheKey, exists := c.Get("idempotency_cache_key"); exists {
		jsonData, _ := json.Marshal(res)
		ctrl.rdb.Set(c.Request.Context(), cacheKey.(string), jsonData, 24*time.Hour)
	}

	response.Success(c, http.StatusCreated, res, nil)
}

func (ctrl *Handler) List(c *gin.Context) {
	userID := c.GetString("user_id_validated")
	status := c.Query("status")
	// Jika Anda ingin defaultnya kosong atau "ALL" agar di SQL nanti jadi NULL
	if status == "ALL" {
		status = ""
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	orders, total, err := ctrl.service.List(c.Request.Context(), userID, status, page, limit)
	if err != nil {
		log.Printf("[Handler.List] Error: %v", err) // Log error service
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	// DEBUG: Cek apakah 'orders' punya isi sebelum dikirim
	log.Printf("[Handler.List] Success fetching %d orders for user %s", len(orders), userID)
	if len(orders) > 0 && len(orders[0].Items) > 0 {
		log.Printf("[Handler.List] First order first item NameSnapshot: %s", orders[0].Items[0].NameSnapshot)
	}

	meta := response.NewPaginationMeta(total, page, limit)

	response.Success(c, http.StatusOK, gin.H{
		"items": orders,
		"meta":  meta,
	}, nil)
}

func (ctrl *Handler) Detail(c *gin.Context) {
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

func (ctrl *Handler) Cancel(c *gin.Context) {
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

func (ctrl *Handler) ListAdmin(c *gin.Context) {
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
func (c *Handler) UpdateStatusByAdmin(ctx *gin.Context) {
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
func (c *Handler) Complete(ctx *gin.Context) {
	id := ctx.Param("id")

	// Ambil UserID dari middleware Auth
	userID := ctx.GetString("user_id_validated")
	if userID == "" {
		response.Error(
			ctx,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		)
		return
	}

	// Langsung paksa status ke COMPLETED karena ini endpoint khusus customer
	res, err := c.service.Complete(ctx.Request.Context(), id, userID, "COMPLETED")
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(200, res)
}
