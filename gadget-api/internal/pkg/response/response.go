package response

import (
	"time"

	"github.com/gin-gonic/gin"
)

type Pagination struct {
	Page            int   `json:"page"`
	PageSize        int   `json:"pageSize"`
	TotalItems      int64 `json:"totalItems"`
	TotalPages      int   `json:"totalPages"`
	HasNextPage     bool  `json:"hasNextPage"`
	HasPreviousPage bool  `json:"hasPreviousPage"`
}

type APIResponse struct {
	Success    bool         `json:"success"`
	Data       interface{}  `json:"data"`
	Pagination *Pagination  `json:"pagination,omitempty"` // omitempty agar tidak muncul jika nil
	Error      *ErrorDetail `json:"error"`
	Message    string       `json:"message"`
	RequestID  string       `json:"requestId"`
	Timestamp  string       `json:"timestamp"`
}

type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details"`
}

// Success untuk data tunggal (tanpa pagination)
func Success(c *gin.Context, status int, message string, data interface{}) {
	requestId := c.GetString("X-Request-ID")
	c.JSON(status, APIResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		RequestID: requestId,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// SuccessWithPagination untuk data list/array
func SuccessWithPagination(c *gin.Context, status int, message string, data interface{}, pag Pagination) {
	requestId := c.GetString("X-Request-ID")
	c.JSON(status, APIResponse{
		Success:    true,
		Data:       data,
		Pagination: &pag,
		Message:    message,
		RequestID:  requestId,
		Timestamp:  time.Now().Format(time.RFC3339),
	})
}

// Error untuk response gagal
func Error(c *gin.Context, status int, errCode string, message string, details interface{}) {
	requestId := c.GetString("X-Request-ID")
	c.JSON(status, APIResponse{
		Success: false,
		Data:    nil,
		Error: &ErrorDetail{
			Code:    errCode,
			Message: message,
			Details: details,
		},
		Message:   message,
		RequestID: requestId,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}
