package cart

import (
	"gadget-api/internal/pkg/response"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	service Service
}

func NewController(s Service) *Controller {
	return &Controller{service: s}
}

func (c *Controller) Create(ctx *gin.Context) {
	userID := ctx.GetString("user_id_validated")
	if err := c.service.Create(ctx, userID); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "CREATE_ERROR", "Gagal membuat cart", err.Error())
		return
	}
	response.Success(ctx, http.StatusCreated, nil, nil)
}

func (c *Controller) AddItem(ctx *gin.Context) {
	userID := ctx.GetString("user_id_validated")
	productID := ctx.Param("productId")

	var req AddItemRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Printf("Failed to parse AddItemRequest: %v", err)

		response.Error(ctx, http.StatusBadRequest, "BAD_REQUEST", "Input tidak valid", err.Error())
		return
	}
	log.Printf("AddItemRequest received: %+v", req)

	// Pastikan productID dari param sinkron dengan req.ProductID jika ada
	req.ProductID = productID

	if err := c.service.AddItem(ctx, userID, req); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "ADD_ITEM_ERROR", "Gagal menambah item ke cart", err.Error())
		return
	}

	response.Success(ctx, http.StatusCreated, nil, nil)
}

func (c *Controller) Count(ctx *gin.Context) {
	userID := ctx.GetString("user_id_validated")

	count, err := c.service.Count(ctx, userID)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, "COUNT_ERROR", "Gagal hitung cart", err.Error())
		return
	}

	response.Success(ctx, http.StatusOK, CartCountResponse{Count: count}, nil)
}

func (c *Controller) Detail(ctx *gin.Context) {
	userID := ctx.GetString("user_id_validated")
	res, err := c.service.Detail(ctx, userID)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, "DETAIL_ERROR", "Gagal mengambil detail cart", err.Error())
		return
	}

	response.Success(ctx, http.StatusOK, res, nil)
}

func (c *Controller) UpdateQty(ctx *gin.Context) {
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

	var req UpdateQtyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(
			ctx,
			http.StatusBadRequest,
			"BAD_REQUEST",
			"Input tidak valid",
			err.Error(),
		)
		return
	}

	if err := c.service.UpdateQty(
		ctx,
		userID,
		ctx.Param("productId"),
		req,
	); err != nil {
		response.Error(
			ctx,
			http.StatusInternalServerError,
			"UPDATE_ERROR",
			"Gagal update quantity",
			err.Error(),
		)
		return
	}

	response.Success(ctx, http.StatusOK, nil, nil)
}

func (c *Controller) Increment(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		response.Error(
			ctx,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		)
		return
	}
	if err := c.service.Increment(ctx, userID.(string), ctx.Param("productId")); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "INCREMENT_ERROR", "Gagal menambah item", err.Error())
		return
	}
	response.Success(ctx, http.StatusOK, nil, nil)
}

func (c *Controller) Decrement(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		response.Error(
			ctx,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		)
		return
	}
	if err := c.service.Decrement(ctx, userID.(string), ctx.Param("productId")); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "DECREMENT_ERROR", "Gagal mengurangi item", err.Error())
		return
	}
	response.Success(ctx, http.StatusOK, nil, nil)
}

func (c *Controller) DeleteItem(ctx *gin.Context) {
	if err := c.service.DeleteItem(ctx, ctx.Param("userId"), ctx.Param("productId")); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "DELETE_ITEM_ERROR", "Gagal menghapus item", err.Error())
		return
	}
	response.Success(ctx, http.StatusOK, nil, nil)
}

func (c *Controller) Delete(ctx *gin.Context) {
	if err := c.service.Delete(ctx, ctx.Param("userId")); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "DELETE_ERROR", "Gagal hapus cart", err.Error())
		return
	}
	response.Success(ctx, http.StatusOK, nil, nil)
}
