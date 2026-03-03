package dashboard

import (
	"go-gadget-api/internal/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) GetDashboard(c *gin.Context) {
	res, err := h.service.GetDashboardData(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	response.Success(c, http.StatusOK, res, &response.PaginationMeta{})
}
