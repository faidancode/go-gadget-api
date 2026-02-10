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

func NewPaginationMeta(total int64, page, limit int) PaginationMeta {
	totalPages := 0
	if limit > 0 {
		// Logika pembulatan ke atas: (total + limit - 1) / limit
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	return PaginationMeta{
		Total:      total,
		TotalPages: totalPages,
		Page:       page,
		PageSize:   limit,
	}
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
