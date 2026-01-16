package product

import (
	"gadget-api/internal/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	service *Service
}

func NewController(s *Service) *Controller {
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
		SortBy:     c.DefaultQuery("sort_by", "newest"), // newest, oldest, price_high, price_low
	}

	// Dukungan untuk route /products/category/:categoryId
	if c.Param("categoryId") != "" {
		req.CategoryID = c.Param("categoryId")
	}

	data, total, err := ctrl.service.ListPublic(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_ERROR", "Gagal mengambil data produk", err.Error())
		return
	}

	response.SuccessWithPagination(c, http.StatusOK, "Berhasil mengambil produk", data, ctrl.makePagination(page, limit, total))
}

// 2. GET ADMIN LIST (Dashboard)
func (ctrl *Controller) GetAdminList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")
	sortCol := c.DefaultQuery("sort_col", "created_at")
	categoryID := c.Query("category_id")

	data, total, err := ctrl.service.ListAdmin(c.Request.Context(), page, limit, search, sortCol, categoryID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_ERROR", "Gagal mengambil data dashboard", err.Error())
		return
	}

	response.SuccessWithPagination(c, http.StatusOK, "Berhasil mengambil dashboard produk", data, ctrl.makePagination(page, limit, total))
}

// 3. CREATE PRODUCT
func (ctrl *Controller) Create(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", err.Error())
		return
	}

	res, err := ctrl.service.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "CREATE_ERROR", err.Error(), nil)
		return
	}
	response.Success(c, http.StatusCreated, "Produk berhasil dibuat", res)
}

// 4. GET BY ID (Admin/Detail)
func (ctrl *Controller) GetByID(c *gin.Context) {
	res, err := ctrl.service.GetByIDAdmin(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "Produk tidak ditemukan", nil)
		return
	}
	response.Success(c, http.StatusOK, "Produk ditemukan", res)
}

func (ctrl *Controller) Update(c *gin.Context) {
	id := c.Param("id")
	var req UpdateProductRequest

	// 1. Bind JSON body
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", err.Error())
		return
	}

	// 2. Panggil Service Update
	// Kita akan membuat fungsi Update di service yang mengembalikan ProductAdminResponse
	res, err := ctrl.service.Update(c.Request.Context(), id, req)
	if err != nil {
		// Menangani error spesifik jika produk tidak ditemukan atau kategori salah
		statusCode := http.StatusInternalServerError
		if err.Error() == "product not found" || err.Error() == "category not found" {
			statusCode = http.StatusNotFound
		}

		response.Error(c, statusCode, "UPDATE_ERROR", "Gagal memperbarui produk", err.Error())
		return
	}

	// 3. Return sukses dengan data terbaru
	response.Success(c, http.StatusOK, "Produk berhasil diperbarui", res)
}

// 5. DELETE & RESTORE
func (ctrl *Controller) Delete(c *gin.Context) {
	if err := ctrl.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		response.Error(c, http.StatusInternalServerError, "DELETE_ERROR", "Gagal menghapus produk", err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Produk berhasil dihapus (soft delete)", nil)
}

func (ctrl *Controller) Restore(c *gin.Context) {
	res, err := ctrl.service.Restore(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "RESTORE_ERROR", "Gagal mengembalikan produk", err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Produk berhasil dikembalikan", res)
}

// Helper untuk membuat objek pagination
func (ctrl *Controller) makePagination(page, limit int, total int64) response.Pagination {
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	return response.Pagination{
		Page:            page,
		PageSize:        limit,
		TotalItems:      total,
		TotalPages:      totalPages,
		HasNextPage:     page < totalPages,
		HasPreviousPage: page > 1,
	}
}
