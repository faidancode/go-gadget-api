package customer

import (
	"fmt"
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

func (h *Handler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := h.bindJSON(c, &req); err != nil {
		return
	}

	customerID := c.GetString("user_id")
	if customerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	res, err := h.service.UpdateProfile(c.Request.Context(), customerID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.DefaultQuery("search", "")
	res, total, err := h.service.ListCustomers(c.Request.Context(), page, limit, search)
	if err != nil {
		h.handleError(c, err)
		return
	}

	meta := response.NewPaginationMeta(total, page, limit)
	response.Success(c, http.StatusOK, res, &meta)
}

func (h *Handler) ToggleStatus(c *gin.Context) {
	id := c.Param("id")
	var req UpdateStatusRequest
	if err := h.bindJSON(c, &req); err != nil {
		return
	}

	res, err := h.service.ToggleCustomerStatus(c.Request.Context(), id, *req.IsActive)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetDetails(c *gin.Context) {
	id := c.Param("id")
	req := CustomerDetailsRequest{
		CustomerID: id,
	}

	res, err := h.service.GetCustomerByID(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

func (h *Handler) GetAddresses(c *gin.Context) {
	id := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.DefaultQuery("search", "")

	req := CustomerAddressesRequest{
		CustomerID: id,
		Page:       page,
		Limit:      limit,
		Search:     search,
	}

	res, err := h.service.ListCustomerAddresses(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res.Data, &res.Meta)
}

func (h *Handler) GetOrders(c *gin.Context) {
	id := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.DefaultQuery("search", "")

	req := CustomerOrdersRequest{
		CustomerID: id,
		Page:       page,
		Limit:      limit,
		Search:     search,
	}

	res, err := h.service.ListCustomerOrders(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res.Data, &res.Meta)
}

func (h *Handler) bindJSON(c *gin.Context, req interface{}) error {
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return err
	}
	return nil
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch err {
	case ErrCustomerNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		fmt.Println("Internal Error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
