package customer

import (
	"fmt"
	"net/http"

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
	res, err := h.service.ListCustomers(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) ToggleStatus(c *gin.Context) {
	id := c.Param("id")
	var req UpdateStatusRequest
	if err := h.bindJSON(c, &req); err != nil {
		return
	}

	res, err := h.service.ToggleCustomerStatus(c.Request.Context(), id, req.IsActive)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetDetails(c *gin.Context) {
	id := c.Param("id")
	res, err := h.service.GetCustomerDetails(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
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
