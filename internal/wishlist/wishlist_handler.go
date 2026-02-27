package wishlist

import (
	"net/http"

	"go-gadget-api/internal/pkg/apperror"
	"go-gadget-api/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	service Service
	logger  *zap.Logger
}

func NewHandler(svc Service, logger ...*zap.Logger) *Handler {
	l := zap.L().Named("order.handler")
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0].Named("order.handler")
	}
	return &Handler{
		service: svc,
		logger:  l,
	}
}

func getUserIDFromContext(c *gin.Context) string {
	if uid := c.GetString("user_id"); uid != "" {
		return uid
	}
	return c.GetString("user_id_validated")
}

// POST /wishlist
func (h *Handler) Create(c *gin.Context) {
	userID := getUserIDFromContext(c)

	h.logger.Debug("http wishlist create request", zap.String("user_id", userID))

	if userID == "" {
		h.logger.Warn("http wishlist create unauthorized: empty userID")
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", nil)
		return
	}

	var req AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("http wishlist create validation failed", zap.Error(err))
		appErr := apperror.Wrap(err, apperror.CodeInvalidInput, "Invalid request body", http.StatusBadRequest)
		httpErr := apperror.ToHTTP(appErr)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, err.Error())
		return
	}

	res, err := h.service.Create(c.Request.Context(), userID, req.ProductID)
	if err != nil {
		h.logger.Error("http wishlist create service error",
			zap.String("user_id", userID),
			zap.String("product_id", req.ProductID),
			zap.Error(err),
		)
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	h.logger.Info("http wishlist item added",
		zap.String("user_id", userID),
		zap.String("product_id", req.ProductID),
	)
	response.Success(c, http.StatusCreated, res, nil)
}

// GET /wishlist
func (h *Handler) List(c *gin.Context) {
	userID := getUserIDFromContext(c)

	h.logger.Debug("http wishlist list request", zap.String("user_id", userID))

	if userID == "" {
		h.logger.Warn("http wishlist list unauthorized")
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", nil)
		return
	}

	res, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("http wishlist list service error", zap.String("user_id", userID), zap.Error(err))
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// DELETE /wishlist/items/:productId
func (h *Handler) Delete(c *gin.Context) {
	userID := getUserIDFromContext(c)
	productID := c.Param("productId")

	h.logger.Debug("http wishlist delete request",
		zap.String("user_id", userID),
		zap.String("product_id", productID),
	)

	if userID == "" {
		h.logger.Warn("http wishlist delete unauthorized")
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", nil)
		return
	}

	if productID == "" {
		h.logger.Warn("http wishlist delete: missing productId param")
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", "productId is required", nil)
		return
	}

	err := h.service.Delete(c.Request.Context(), userID, productID)
	if err != nil {
		h.logger.Error("http wishlist delete service error",
			zap.String("user_id", userID),
			zap.String("product_id", productID),
			zap.Error(err),
		)
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	h.logger.Info("http wishlist item removed",
		zap.String("user_id", userID),
		zap.String("product_id", productID),
	)
	response.Success(c, http.StatusOK, gin.H{
		"message": "Product removed from wishlist successfully",
	}, nil)
}
