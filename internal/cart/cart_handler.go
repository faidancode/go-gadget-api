package cart

import (
	"net/http"

	"go-gadget-api/internal/pkg/response"
	"go-gadget-api/internal/shared/contextutil"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	service Service
	logger  *zap.Logger
}

func NewHandler(svc Service, logger ...*zap.Logger) *Handler {
	l := zap.L().Named("cart.handler")
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0].Named("cart.handler")
	}
	return &Handler{
		service: svc,
		logger:  l,
	}
}

func getUserIDFromContext(ctx *gin.Context) string {
	if uid := ctx.GetString("user_id"); uid != "" {
		return uid
	}
	return ctx.GetString("user_id_validated")
}

func (h *Handler) Create(ctx *gin.Context) {
	logger := contextutil.GetLogger(ctx.Request.Context(), h.logger)
	userID := getUserIDFromContext(ctx)

	if err := h.service.Create(ctx.Request.Context(), userID); err != nil {
		logger.Error("http cart create failed", zap.Error(err))
		response.Error(ctx, http.StatusInternalServerError, "CREATE_ERROR", "Gagal membuat cart", err.Error())
		return
	}
	response.Success(ctx, http.StatusCreated, nil, nil)
}

func (h *Handler) AddItem(ctx *gin.Context) {
	logger := contextutil.GetLogger(ctx.Request.Context(), h.logger)
	userID := getUserIDFromContext(ctx)
	productID := ctx.Param("productId")

	var req AddItemRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.Warn("http cart add item validation failed", zap.Error(err))
		response.Error(ctx, http.StatusBadRequest, "BAD_REQUEST", "Input tidak valid", err.Error())
		return
	}

	req.ProductID = productID
	logger.Debug("adding item to cart", zap.String("product_id", productID))

	if err := h.service.AddItem(ctx.Request.Context(), userID, req); err != nil {
		logger.Error("http cart add item service failed", zap.Error(err))
		response.Error(ctx, http.StatusInternalServerError, "ADD_ITEM_ERROR", "Gagal menambah item ke cart", err.Error())
		return
	}

	response.Success(ctx, http.StatusCreated, nil, nil)
}

func (h *Handler) Count(ctx *gin.Context) {
	logger := contextutil.GetLogger(ctx.Request.Context(), h.logger)
	userID := getUserIDFromContext(ctx)

	count, err := h.service.Count(ctx.Request.Context(), userID)
	if err != nil {
		logger.Error("http cart count service failed", zap.Error(err))
		response.Error(ctx, http.StatusInternalServerError, "COUNT_ERROR", "Gagal hitung cart", err.Error())
		return
	}

	response.Success(ctx, http.StatusOK, CartCountResponse{Count: count}, nil)
}

func (h *Handler) Detail(ctx *gin.Context) {
	logger := contextutil.GetLogger(ctx.Request.Context(), h.logger)
	userID := getUserIDFromContext(ctx)

	res, err := h.service.Detail(ctx.Request.Context(), userID)
	if err != nil {
		logger.Error("http cart detail service failed", zap.Error(err))
		response.Error(ctx, http.StatusInternalServerError, "DETAIL_ERROR", "Gagal mengambil detail cart", err.Error())
		return
	}

	response.Success(ctx, http.StatusOK, res, nil)
}

func (h *Handler) UpdateQty(ctx *gin.Context) {
	logger := contextutil.GetLogger(ctx.Request.Context(), h.logger)
	userID := getUserIDFromContext(ctx)
	productID := ctx.Param("productId")

	var req UpdateQtyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.Warn("http cart update qty validation failed", zap.Error(err))
		response.Error(ctx, http.StatusBadRequest, "BAD_REQUEST", "Input tidak valid", err.Error())
		return
	}

	if err := h.service.UpdateQty(ctx.Request.Context(), userID, productID, req); err != nil {
		logger.Error("http cart update qty service failed", zap.String("product_id", productID), zap.Error(err))
		response.Error(ctx, http.StatusInternalServerError, "UPDATE_ERROR", "Gagal update quantity", err.Error())
		return
	}

	response.Success(ctx, http.StatusOK, nil, nil)
}

func (h *Handler) Increment(ctx *gin.Context) {
	logger := contextutil.GetLogger(ctx.Request.Context(), h.logger)
	userID := getUserIDFromContext(ctx)
	productID := ctx.Param("productId")

	if err := h.service.Increment(ctx.Request.Context(), userID, productID); err != nil {
		logger.Error("http cart increment failed", zap.String("product_id", productID), zap.Error(err))
		response.Error(ctx, http.StatusInternalServerError, "INCREMENT_ERROR", "Gagal menambah item", err.Error())
		return
	}
	response.Success(ctx, http.StatusOK, nil, nil)
}

func (h *Handler) Decrement(ctx *gin.Context) {
	logger := contextutil.GetLogger(ctx.Request.Context(), h.logger)
	userID := getUserIDFromContext(ctx)
	productID := ctx.Param("productId")

	if err := h.service.Decrement(ctx.Request.Context(), userID, productID); err != nil {
		logger.Error("http cart decrement failed", zap.String("product_id", productID), zap.Error(err))
		response.Error(ctx, http.StatusInternalServerError, "DECREMENT_ERROR", "Gagal mengurangi item", err.Error())
		return
	}
	response.Success(ctx, http.StatusOK, nil, nil)
}

func (h *Handler) DeleteItem(ctx *gin.Context) {
	logger := contextutil.GetLogger(ctx.Request.Context(), h.logger)
	userID := getUserIDFromContext(ctx)
	productID := ctx.Param("productId")

	if err := h.service.DeleteItem(ctx.Request.Context(), userID, productID); err != nil {
		logger.Error("http cart delete item failed", zap.String("product_id", productID), zap.Error(err))
		response.Error(ctx, http.StatusInternalServerError, "DELETE_ITEM_ERROR", "Gagal menghapus item", err.Error())
		return
	}

	response.Success(ctx, http.StatusOK, nil, nil)
}

func (h *Handler) ClearCart(ctx *gin.Context) {
	logger := contextutil.GetLogger(ctx.Request.Context(), h.logger)
	userID := getUserIDFromContext(ctx)

	if err := h.service.ClearCart(ctx.Request.Context(), userID); err != nil {
		logger.Error("http cart clear failed", zap.Error(err))
		response.Error(ctx, http.StatusInternalServerError, "CLEAR_CART_ERROR", "Gagal menghapus cart", err.Error())
		return
	}

	response.Success(ctx, http.StatusOK, nil, nil)
}
