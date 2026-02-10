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

func (ctrl *Handler) UpdateProfile(c *gin.Context) {
	// Menggunakan user_id_validated dari middleware
	customerID := c.GetString("user_id_validated")
	if customerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	res, err := ctrl.service.UpdateProfile(
		c.Request.Context(),
		customerID,
		req,
	)

	if err != nil {
		switch err {
		case ErrCustomerNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			// Log error asli di sini untuk internal tracing
			fmt.Println("Internal Error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, res)
}
