package category

import (
	"gadget-api/internal/pkg/response"
	"gadget-api/internal/pkg/utils"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 10
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
	id := c.Param("id")

	// ✅ Validasi UUID di controller
	if _, err := uuid.Parse(id); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			"INVALID_ID",
			"Invalid category ID",
			err.Error(),
		)
		return
	}

	res, err := ctrl.service.GetByID(c.Request.Context(), id)
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

// 3. CREATE BRAND
func (ctrl *Controller) Create(c *gin.Context) {
	ctx := c.Request.Context()

	// 1. Parse multipart form (max 10 MB)
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			"INVALID_FORM",
			"Invalid multipart form",
			err.Error(),
		)
		return
	}

	// 2. Parse form values
	name := c.PostForm("name")
	description := c.PostForm("description")

	req := CreateCategoryRequest{
		Name:        name,
		Slug:        utils.GenerateSlug(name),
		Description: description,
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
	log.Println(req)

	// 4. Handle optional file upload
	var (
		file     multipart.File
		filename string
	)

	fileHeader, err := c.FormFile("image") // form-data key: image
	if err == nil && fileHeader != nil {
		openedFile, err := fileHeader.Open()
		if err != nil {
			response.Error(
				c,
				http.StatusBadRequest,
				"FILE_ERROR",
				"Failed to open uploaded file",
				err.Error(),
			)
			return
		}

		defer openedFile.Close()

		file = openedFile
		filename = fileHeader.Filename
	}

	// 5. Call service
	result, err := ctrl.service.Create(ctx, req, file, filename)
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

	// 6. Success response
	response.Success(
		c,
		http.StatusCreated,
		result,
		nil,
	)
}

func (ctrl *Controller) Update(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			"INVALID_ID",
			"Invalid ID format",
			err.Error(),
		)
		return
	}

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
	// 2. Parse form values
	name := c.PostForm("name")
	description := c.PostForm("description")

	// 2. Parse form fields (sesuaikan dengan struct UpdateCategoryRequest Anda)
	// Karena multipart/form-data, kita ambil via PostForm, bukan BindJSON
	req := UpdateCategoryRequest{
		Name:        name,
		Slug:        utils.GenerateSlug(name),
		Description: description,
	}

	// 3. Get uploaded file (optional)
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

	// 4. Call service
	// Pastikan Service.Update sudah diupdate signature-nya untuk menerima (ctx, id, req, file, filename)
	res, err := ctrl.service.Update(c.Request.Context(), c.Param("id"), req, file, filename)
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
	id := c.Param("id")

	// 1. Validasi UUID lebih awal
	if _, err := uuid.Parse(id); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			"INVALID_ID",
			"Invalid category ID",
			err.Error(),
		)
		return
	}

	// 2. Panggil service
	if err := ctrl.service.Delete(c.Request.Context(), id); err != nil {
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
