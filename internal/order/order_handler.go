package order

import (
	"encoding/json"
	"go-gadget-api/internal/pkg/apperror"
	"go-gadget-api/internal/pkg/response"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Handler struct {
	service Service
	rdb     *redis.Client
	logger  *zap.Logger
}

func NewHandler(svc Service, rdb *redis.Client, logger ...*zap.Logger) *Handler {
	l := zap.L().Named("order.handler")
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0].Named("order.handler")
	}
	return &Handler{service: svc, rdb: rdb, logger: l}
}

// ==================== CUSTOMER ENDPOINTS ====================

// Checkout creates a new order from user's cart
// POST /orders
func (h *Handler) Checkout(c *gin.Context) {
	// Mengambil userID dari context (disetel oleh AuthMiddleware)
	userID := c.GetString("user_id_validated") // Gunakan key yang konsisten dengan middleware Anda

	h.logger.Debug("http checkout request",
		zap.String("user_id", userID),
	)

	if userID == "" {
		h.logger.Warn("http checkout unauthorized: empty userID")
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", nil)
		return
	}

	// Idempotency Lock Key
	lockKey, _ := c.Get("idempotency_lock_key")
	defer func() {
		if lockKey != nil {
			h.rdb.Del(c.Request.Context(), lockKey.(string))
		}
	}()

	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("http checkout validation failed", zap.Error(err))
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", err.Error())
		return
	}

	// Memanggil Service
	res, err := h.service.Checkout(c.Request.Context(), userID, req)
	if err != nil {
		h.logger.Error("http checkout service error",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		// Menggunakan helper writeServiceError jika ada di struct handler Anda
		// Jika tidak, Anda bisa menggunakan apperror.ToHTTP(err) secara manual
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	// Simpan hasil ke cache Idempotency jika sukses
	if cacheKey, exists := c.Get("idempotency_cache_key"); exists {
		jsonData, _ := json.Marshal(res)
		h.rdb.Set(c.Request.Context(), cacheKey.(string), jsonData, 24*time.Hour)

		h.logger.Debug("idempotency response cached",
			zap.String("cache_key", cacheKey.(string)),
		)
	}

	response.Success(c, http.StatusCreated, res, nil)
}

func (h *Handler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	status := c.Query("status")
	// Jika Anda ingin defaultnya kosong atau "ALL" agar di SQL nanti jadi NULL
	if status == "ALL" {
		status = ""
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	orders, total, err := h.service.List(c.Request.Context(), userID, status, page, limit)
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

func (h *Handler) Detail(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		httpErr := apperror.ToHTTP(ErrInvalidOrderID)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	res, err := h.service.Detail(c.Request.Context(), orderID)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

func (h *Handler) Cancel(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		httpErr := apperror.ToHTTP(ErrInvalidOrderID)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	if err := h.service.Cancel(c.Request.Context(), orderID); err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Order cancelled successfully",
	}, nil)
}

// ==================== ADMIN ENDPOINTS ====================

func (h *Handler) ListAdmin(c *gin.Context) {
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

	orders, total, err := h.service.ListAdmin(
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
	userID := ctx.GetString("user_id")
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
