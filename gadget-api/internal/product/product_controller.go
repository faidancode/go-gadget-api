package product

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

// 1. GET PUBLIC LIST (Customers)
func (ctrl *Controller) GetPublicList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	minPrice, _ := strconv.ParseFloat(c.DefaultQuery("min_price", "0"), 64)
	maxPrice, _ := strconv.ParseFloat(c.DefaultQuery("max_price", "0"), 64)

	req := ListPublicRequest{
		Page:       page,
		Limit:      limit,
		Search:     c.Query("search"),
		CategoryID: c.Query("category_id"),
		MinPrice:   minPrice,
		MaxPrice:   maxPrice,
		SortBy:     c.DefaultQuery("sort_by", "newest"),
	}

	// Support route: /products/category/:categoryId
	if c.Param("categoryId") != "" {
		req.CategoryID = c.Param("categoryId")
	}

	data, total, err := ctrl.service.ListPublic(c.Request.Context(), req)
	if err != nil {
		response.Error(
			c,
			http.StatusInternalServerError,
			"FETCH_ERROR",
			"Gagal mengambil data produk",
			err.Error(),
		)
		return
	}

	response.Success(
		c,
		http.StatusOK,
		data,
		ctrl.makePagination(page, limit, total),
	)
}

// 2. GET ADMIN LIST (Dashboard)
func (ctrl *Controller) GetAdminList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	search := c.Query("search")
	sortCol := c.DefaultQuery("sort_col", "created_at")
	categoryID := c.Query("category_id")

	data, total, err := ctrl.service.ListAdmin(
		c.Request.Context(),
		page,
		limit,
		search,
		sortCol,
		categoryID,
	)
	if err != nil {
		response.Error(
			c,
			http.StatusInternalServerError,
			"FETCH_ERROR",
			"Gagal mengambil data dashboard produk",
			err.Error(),
		)
		return
	}

	response.Success(
		c,
		http.StatusOK,
		data,
		ctrl.makePagination(page, limit, total),
	)
}

// 3. CREATE PRODUCT
func (ctrl *Controller) Create(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Input tidak valid",
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
			"Gagal membuat produk",
			err.Error(),
		)
		return
	}

	response.Success(c, http.StatusCreated, res, nil)
}

// 4. GET BY ID (Admin / Detail)
func (ctrl *Controller) GetByID(c *gin.Context) {
	res, err := ctrl.service.GetByIDAdmin(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(
			c,
			http.StatusNotFound,
			"NOT_FOUND",
			"Produk tidak ditemukan",
			nil,
		)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// 5. UPDATE PRODUCT
func (ctrl *Controller) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Input tidak valid",
			err.Error(),
		)
		return
	}

	res, err := ctrl.service.Update(c.Request.Context(), id, req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "product not found" || err.Error() == "category not found" {
			statusCode = http.StatusNotFound
		}

		response.Error(
			c,
			statusCode,
			"UPDATE_ERROR",
			"Gagal memperbarui produk",
			err.Error(),
		)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// 6. DELETE PRODUCT (Soft Delete)
func (ctrl *Controller) Delete(c *gin.Context) {
	if err := ctrl.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		response.Error(
			c,
			http.StatusInternalServerError,
			"DELETE_ERROR",
			"Gagal menghapus produk",
			err.Error(),
		)
		return
	}

	response.Success(c, http.StatusOK, nil, nil)
}

// 7. RESTORE PRODUCT
func (ctrl *Controller) Restore(c *gin.Context) {
	res, err := ctrl.service.Restore(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(
			c,
			http.StatusInternalServerError,
			"RESTORE_ERROR",
			"Gagal mengembalikan produk",
			err.Error(),
		)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// Helper: Pagination Meta
func (ctrl *Controller) makePagination(page, limit int, total int64) *response.PaginationMeta {
	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	return &response.PaginationMeta{
		Total:      total,
		TotalPages: totalPages,
		Page:       page,
		PageSize:   limit,
	}
}
