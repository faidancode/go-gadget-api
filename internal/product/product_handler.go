package product

import (
	"context"
	"fmt"
	"go-gadget-api/internal/pkg/apperror"
	"go-gadget-api/internal/pkg/httpx"
	"go-gadget-api/internal/pkg/response"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ReviewService interface {
	CheckEligibility(
		ctx context.Context,
		userID string,
		productSlug string,
	) (EligibilityResponse, error)
}

type Handler struct {
	productService Service
	reviewService  ReviewService
}

func NewHandler(
	productService Service,
	reviewService ReviewService,
) *Handler {
	return &Handler{
		productService: productService,
		reviewService:  reviewService,
	}
}

// 1. GET PUBLIC LIST (Customers)
func (h *Handler) GetPublicList(c *gin.Context) {
	var q ListPublicQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_QUERY", "Query tidak valid", err.Error())
		return
	}

	brandSlug := q.BrandSlug
	if brandSlug == "" {
		brandSlug = c.Query("brand_slug")
	}

	req := ListPublicRequest{
		Page:        q.Page,
		Limit:       q.Limit,
		Search:      q.Search,
		BrandSlug:   brandSlug,
		CategoryIDs: q.CategoryIDs,
		MinPrice:    q.MinPrice,
		MaxPrice:    q.MaxPrice,
		SortBy:      q.SortBy,
	}

	data, total, err := h.productService.ListPublic(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_ERROR", "Gagal mengambil data produk", err.Error())
		return
	}

	response.Success(c, http.StatusOK, data, h.makePagination(q.Page, q.Limit, total))
}

// 2. GET ADMIN LIST (Dashboard)
func (h *Handler) GetAdminList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	sort := httpx.ParseSort(c, "created_at", "desc")

	req := ListProductAdminRequest{
		Page:     page,
		Limit:    limit,
		Search:   c.Query("search"),
		Category: c.Query("category_id"),
		SortBy:   sort.SortBy,
		SortDir:  sort.SortDir,
	}

	data, total, err := h.productService.ListAdmin(c.Request.Context(), req)
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
		h.makePagination(page, limit, total),
	)
}

func (h *Handler) Create(c *gin.Context) {
	// 1. Parse multipart form (max 10 MB)
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_FORM", "Invalid multipart form", err.Error())
		return
	}

	// 2. Parse form values secara MANUAL (Seperti di Category)
	var price float64
	fmt.Sscanf(c.PostForm("price"), "%f", &price)

	var stock int32
	fmt.Sscanf(c.PostForm("stock"), "%d", &stock)

	brandID := c.PostForm("brandId")
	if brandID == "" {
		brandID = c.PostForm("brand_id")
	}

	categoryID := c.PostForm("categoryId")
	if categoryID == "" {
		categoryID = c.PostForm("category_id")
	}

	req := CreateProductRequest{
		BrandID:     brandID,
		CategoryID:  categoryID,
		Name:        c.PostForm("name"),
		Description: c.PostForm("description"),
		SKU:         c.PostForm("sku"),
		Price:       price,
		Stock:       stock,
	}

	// Debug log setelah diisi manual
	log.Printf("Received CreateProductRequest: %+v", req)

	// 3. Handle optional file upload
	var (
		file     multipart.File
		filename string
	)
	fileHeader, err := c.FormFile("image")
	if err == nil && fileHeader != nil {
		openedFile, err := fileHeader.Open()
		if err != nil {
			response.Error(c, http.StatusBadRequest, "FILE_ERROR", "Failed to open file", err.Error())
			return
		}
		defer openedFile.Close()
		file = openedFile
		filename = fileHeader.Filename
	}

	// 4. Call service
	result, err := h.productService.Create(c.Request.Context(), req, file, filename)
	if err != nil {
		// Gunakan helper apperror.ToHTTP agar error mapping Anda berjalan
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, result, nil)
}

// 4. GET BY ID (Admin / Detail)
func (h *Handler) GetByID(c *gin.Context) {
	res, err := h.productService.GetByID(c.Request.Context(), c.Param("id"))
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

func (h *Handler) GetBySlug(c *gin.Context) {
	res, err := h.productService.GetBySlug(c.Request.Context(), c.Param("slug"))
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

func (h *Handler) CheckReviewEligibility(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)
	productSlug := c.Param("slug")

	res, err := h.reviewService.CheckEligibility(c.Request.Context(), userIDStr, productSlug)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// 5. UPDATE PRODUCT
func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")

	// 1. Parse multipart form
	err := c.Request.ParseMultipartForm(10 << 20) // 10 MB max
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

	// 2. Parse form fields (all optional for update)
	brandID := c.PostForm("brandId")
	if brandID == "" {
		brandID = c.PostForm("brand_id")
	}
	categoryID := c.PostForm("categoryId")
	if categoryID == "" {
		categoryID = c.PostForm("category_id")
	}

	req := UpdateProductRequest{
		BrandID:     brandID,
		CategoryID:  categoryID,
		Name:        c.PostForm("name"),
		Description: c.PostForm("description"),
		SKU:         c.PostForm("sku"),
	}

	// Parse numeric fields
	if priceStr := c.PostForm("price"); priceStr != "" {
		var price float64
		_, err := fmt.Sscanf(priceStr, "%f", &price)
		if err == nil {
			req.Price = price
		}
	}

	if stockStr := c.PostForm("stock"); stockStr != "" {
		var stock int32
		_, err := fmt.Sscanf(stockStr, "%d", &stock)
		if err == nil {
			req.Stock = stock
		}
	}

	isActiveStr := c.PostForm("isActive")
	if isActiveStr == "" {
		isActiveStr = c.PostForm("is_active")
	}
	if isActiveStr != "" {
		isActive := isActiveStr == "true"
		req.IsActive = &isActive
	}

	// 3. Get uploaded file (optional)
	var file multipart.File
	var filename string
	fileHeader, err := c.FormFile("image")
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
	res, err := h.productService.Update(c.Request.Context(), id, req, file, filename)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// 6. DELETE PRODUCT (Soft Delete)
func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.productService.Delete(c.Request.Context(), id)
	if err != nil {
		// Mapping error dari service ke HTTP error
		// Jika err adalah ErrProductNotFound, mapping ini harus menghasilkan status 404
		httpErr := apperror.ToHTTP(err)

		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, nil, nil)
}

// 7. RESTORE PRODUCT
func (h *Handler) Restore(c *gin.Context) {
	res, err := h.productService.Restore(c.Request.Context(), c.Param("id"))
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
func (h *Handler) makePagination(page, limit int, total int64) *response.PaginationMeta {
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
