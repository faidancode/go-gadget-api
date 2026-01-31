package response

import (
	"github.com/gin-gonic/gin"
)

type PaginationMeta struct {
	Total      int64 `json:"total,omitempty"`
	TotalPages int   `json:"totalPages,omitempty"`
	Page       int   `json:"page,omitempty"`
	PageSize   int   `json:"pageSize,omitempty"`
}

type ApiEnvelope struct {
	Ok    bool                   `json:"ok"`
	Data  interface{}            `json:"data"`
	Meta  *PaginationMeta        `json:"meta"`
	Error map[string]interface{} `json:"error"`
}

func Success(c *gin.Context, status int, data interface{}, meta *PaginationMeta) {
	c.JSON(status, ApiEnvelope{
		Ok:    true,
		Data:  data,
		Meta:  meta,
		Error: nil,
	})
}

func Error(c *gin.Context, status int, errorCode string, message string, details interface{}) {
	c.JSON(status, ApiEnvelope{
		Ok:   false,
		Data: nil,
		Meta: nil,
		Error: map[string]interface{}{
			"code":    errorCode,
			"message": message,
			"details": details,
		},
	})
}
