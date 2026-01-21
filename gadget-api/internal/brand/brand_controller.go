package brand

import (
	"gadget-api/internal/pkg/response"
	"mime/multipart"
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
	var req ListBrandRequest

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
			"Brand not found",
			nil,
		)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// 3. CREATE BRAND
func (ctrl *Controller) Create(c *gin.Context) {
	// 1. Parse multipart form (Max 10 MB)
	err := c.Request.ParseMultipartForm(10 << 20)
	if err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			"INVALID_FORM",
			"Invalid multipart form",
			err.Error(),
		)
		return
	}

	// 2. Parse form fields
	req := CreateBrandRequest{
		Name:        c.PostForm("name"),
		Description: c.PostForm("description"),
	}

	// 3. Validate required fields
	if req.Name == "" {
		response.Error(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Missing required fields: name",
			nil,
		)
		return
	}

	// 4. Get uploaded file (optional)
	var file multipart.File
	var filename string
	fileHeader, err := c.FormFile("image") // Key form-data: "image"
	if err == nil && fileHeader != nil {
		file, err = fileHeader.Open()
		if err != nil {
			response.Error(c, http.StatusBadRequest, "FILE_ERROR", "Failed to open uploaded file", err.Error())
			return
		}
		defer file.Close()
		filename = fileHeader.Filename
	}

	// 5. Call service (Pastikan signature Service.Create sudah diupdate untuk menerima file)
	res, err := ctrl.service.Create(c.Request.Context(), req, file, filename)
	if err != nil {
		// Menggunakan error mapper jika ada, jika tidak gunakan error default
		response.Error(
			c,
			http.StatusInternalServerError,
			"CREATE_ERROR",
			"Failed to create brand",
			err.Error(),
		)
		return
	}

	response.Success(c, http.StatusCreated, res, nil)
}

func (ctrl *Controller) Update(c *gin.Context) {
	var req CreateBrandRequest
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
			"Failed to update brand",
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
			"Failed to delete brand",
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
			"Failed to restore brand",
			err.Error(),
		)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}
