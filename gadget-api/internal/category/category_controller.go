package category

import (
	"gadget-api/internal/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	service Service
}

func NewController(s Service) *Controller {
	return &Controller{service: s}
}

func (ctrl *Controller) ListPublic(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	data, total, err := ctrl.service.ListPublic(c.Request.Context(), page, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_ERROR", "Gagal mengambil kategori", err.Error())
		return
	}

	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	response.Success(c, http.StatusOK, data, &response.PaginationMeta{
		Total:      total,
		TotalPages: totalPages,
		Page:       page,
		PageSize:   limit,
	})
}

func (ctrl *Controller) ListAdmin(c *gin.Context) {
	var req ListCategoryRequest

	// Bind query parameters ke struct (page, limit, search, sort_col, sort_dir)
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", "Parameter pencarian tidak valid", err.Error())
		return
	}

	// Memanggil service dengan struct req
	data, total, err := ctrl.service.ListAdmin(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_ERROR", "Gagal mengambil daftar kategori admin", err.Error())
		return
	}

	// Kalkulasi Pagination
	totalPages := 0
	if req.Limit > 0 {
		totalPages = int((total + int64(req.Limit) - 1) / int64(req.Limit))
	}

	response.Success(c, http.StatusOK, data, &response.PaginationMeta{
		Total:      total,
		TotalPages: totalPages,
		Page:       int(req.Page),
		PageSize:   int(req.Limit),
	})
}

func (ctrl *Controller) GetByID(c *gin.Context) {
	res, err := ctrl.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(
			c,
			http.StatusNotFound,
			"NOT_FOUND",
			"Category not found",
			nil,
		)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

func (ctrl *Controller) Create(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Invalid input",
			err.Error(),
		)
		return
	}

	res, err := ctrl.service.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(
			c,
			http.StatusInternalServerError,
			"CREATE_ERROR",
			"Failed to create category",
			err.Error(),
		)
		return
	}

	response.Success(c, http.StatusCreated, res, nil)
}

func (ctrl *Controller) Update(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Invalid input",
			err.Error(),
		)
		return
	}

	res, err := ctrl.service.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(
			c,
			http.StatusInternalServerError,
			"UPDATE_ERROR",
			"Failed to update category",
			err.Error(),
		)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

func (ctrl *Controller) Delete(c *gin.Context) {
	if err := ctrl.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		response.Error(
			c,
			http.StatusInternalServerError,
			"DELETE_ERROR",
			"Failed to delete category",
			err.Error(),
		)
		return
	}

	response.Success(c, http.StatusOK, nil, nil)
}

func (ctrl *Controller) Restore(c *gin.Context) {
	res, err := ctrl.service.Restore(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(
			c,
			http.StatusInternalServerError,
			"RESTORE_ERROR",
			"Failed to restore category",
			err.Error(),
		)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}
