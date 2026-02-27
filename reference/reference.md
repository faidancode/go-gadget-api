## Repo

package product

import (
	"context"
	"gadget-api/internal/dbgen"

	"github.com/google/uuid"
)

//go:generate mockgen -source=product_repo.go -destination=mock/product_repo_mock.go -package=mock
type Repository interface {
	Create(ctx context.Context, arg dbgen.CreateProductParams) (dbgen.Product, error)
	// Pisahkan List menjadi Public dan Admin sesuai query.sql terbaru
	ListPublic(ctx context.Context, arg dbgen.ListProductsPublicParams) ([]dbgen.ListProductsPublicRow, error)
	ListAdmin(ctx context.Context, arg dbgen.ListProductsAdminParams) ([]dbgen.ListProductsAdminRow, error)

	GetByID(ctx context.Context, id uuid.UUID) (dbgen.GetProductByIDRow, error)
	Update(ctx context.Context, arg dbgen.UpdateProductParams) (dbgen.Product, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) (dbgen.Product, error)
}

type repository struct {
	queries *dbgen.Queries
}

func NewRepository(q *dbgen.Queries) Repository {
	return &repository{queries: q}
}

func (r *repository) Create(ctx context.Context, arg dbgen.CreateProductParams) (dbgen.Product, error) {
	return r.queries.CreateProduct(ctx, arg)
}

// Implementasi List untuk Customer (Hanya barang aktif & filter harga/sort)
func (r *repository) ListPublic(ctx context.Context, arg dbgen.ListProductsPublicParams) ([]dbgen.ListProductsPublicRow, error) {
	return r.queries.ListProductsPublic(ctx, arg)
}

// Implementasi List untuk Admin (Semua barang & filter dashboard)
func (r *repository) ListAdmin(ctx context.Context, arg dbgen.ListProductsAdminParams) ([]dbgen.ListProductsAdminRow, error) {
	return r.queries.ListProductsAdmin(ctx, arg)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (dbgen.GetProductByIDRow, error) {
	return r.queries.GetProductByID(ctx, id)
}

func (r *repository) Update(ctx context.Context, arg dbgen.UpdateProductParams) (dbgen.Product, error) {
	return r.queries.UpdateProduct(ctx, arg)
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.SoftDeleteProduct(ctx, id)
}

func (r *repository) Restore(ctx context.Context, id uuid.UUID) (dbgen.Product, error) {
	return r.queries.RestoreProduct(ctx, id)
}


## Service
package product

import (
	"context"
	"database/sql"
	"fmt"
	"gadget-api/internal/category"
	"gadget-api/internal/dbgen"
	"strconv"
	"strings"

	"github.com/google/uuid"
)


type Service struct {
	repo    Repository
	catRepo category.Repository
}

func NewService(repo Repository, catRepo category.Repository) *Service {
	return &Service{
		repo:    repo,
		catRepo: catRepo,
	}
}

func (s *Service) ListPublic(ctx context.Context, req ListPublicRequest) ([]ProductResponse, int64, error) {
	offset := (req.Page - 1) * req.Limit

	if req.MaxPrice == 0 {
		req.MaxPrice = 999999999
	}

	params := dbgen.ListProductsPublicParams{
		Limit:    int32(req.Limit),
		Offset:   int32(offset),
		Search:   dbgen.NewNullString(req.Search),
		MinPrice: fmt.Sprintf("%.2f", req.MinPrice),
		MaxPrice: fmt.Sprintf("%.2f", req.MaxPrice),
		SortBy:   req.SortBy,
	}

	if req.CategoryID != "" {
		uid, err := uuid.Parse(req.CategoryID)
		if err == nil {
			params.CategoryID = uuid.NullUUID{UUID: uid, Valid: true}
		}
	}

	rows, err := s.repo.ListPublic(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	return s.mapToPublicResponse(rows)
}

func (s *Service) ListAdmin(ctx context.Context, page, limit int, search, sortCol, categoryID string) ([]ProductAdminResponse, int64, error) {
	offset := (page - 1) * limit

	params := dbgen.ListProductsAdminParams{
		Limit:   int32(limit),
		Offset:  int32(offset),
		Search:  dbgen.NewNullString(search),
		SortCol: sortCol,
	}

	if categoryID != "" {
		uid, err := uuid.Parse(categoryID)
		if err == nil {
			params.CategoryID = uuid.NullUUID{UUID: uid, Valid: true}
		}
	}

	rows, err := s.repo.ListAdmin(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	return s.mapToAdminResponse(rows)
}

func (s *Service) Create(ctx context.Context, req CreateProductRequest) (ProductAdminResponse, error) {
	catID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return ProductAdminResponse{}, fmt.Errorf("invalid category id")
	}

	_, err = s.catRepo.GetByID(ctx, catID)
	if err != nil {
		return ProductAdminResponse{}, fmt.Errorf("category not found")
	}

	slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-")) + "-" + uuid.New().String()[:5]
	priceStr := fmt.Sprintf("%.2f", req.Price)

	p, err := s.repo.Create(ctx, dbgen.CreateProductParams{
		CategoryID:  catID,
		Name:        req.Name,
		Slug:        slug,
		Description: dbgen.NewNullString(req.Description),
		Price:       priceStr,
		Stock:       req.Stock,
		Sku:         dbgen.NewNullString(req.SKU),
		ImageUrl:    dbgen.NewNullString(req.ImageUrl),
	})

	if err != nil {
		return ProductAdminResponse{}, err
	}

	return s.GetByIDAdmin(ctx, p.ID.String())
}

func (s *Service) GetByIDAdmin(ctx context.Context, idStr string) (ProductAdminResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return ProductAdminResponse{}, fmt.Errorf("invalid product id")
	}

	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ProductAdminResponse{}, err
	}

	priceFloat, _ := strconv.ParseFloat(p.Price, 64)
	return ProductAdminResponse{
		ID:           p.ID.String(),
		CategoryName: p.CategoryName,
		Name:         p.Name,
		Slug:         p.Slug,
		Price:        priceFloat,
		Stock:        p.Stock,
		SKU:          p.Sku.String,
		IsActive:     p.IsActive.Bool,
		CreatedAt:    p.CreatedAt,
	}, nil
}

func (s *Service) Update(ctx context.Context, idStr string, req UpdateProductRequest) (ProductAdminResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return ProductAdminResponse{}, fmt.Errorf("invalid product id")
	}

	// 1. Cek apakah produk ada
	existingProduct, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ProductAdminResponse{}, fmt.Errorf("product not found")
	}

	// 2. Siapkan params untuk repo
	params := dbgen.UpdateProductParams{
		ID:          id,
		Name:        existingProduct.Name,
		Description: existingProduct.Description,
		Price:       existingProduct.Price,
		Stock:       existingProduct.Stock,
		Sku:         existingProduct.Sku,
		ImageUrl:    existingProduct.ImageUrl,
		CategoryID:  existingProduct.CategoryID,
		IsActive:    existingProduct.IsActive,
	}

	// 3. Update hanya field yang dikirim (Patch-like behavior)
	if req.Name != "" {
		params.Name = req.Name
	}
	if req.CategoryID != "" {
		catID, err := uuid.Parse(req.CategoryID)
		if err == nil {
			params.CategoryID = catID
		}
	}
	if req.Price > 0 {
		params.Price = fmt.Sprintf("%.2f", req.Price)
	}
	if req.Stock != 0 {
		params.Stock = req.Stock
	}
	if req.SKU != "" {
		params.Sku = dbgen.NewNullString(req.SKU)
	}
	if req.Description != "" {
		params.Description = dbgen.NewNullString(req.Description)
	}
	if req.IsActive != nil {
		params.IsActive = sql.NullBool{Bool: *req.IsActive, Valid: true}
	}

	// 4. Eksekusi Update ke Repo
	_, err = s.repo.Update(ctx, params)
	if err != nil {
		return ProductAdminResponse{}, err
	}

	// 5. Kembalikan data terbaru
	return s.GetByIDAdmin(ctx, idStr)
}

func (s *Service) Delete(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid product id")
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) Restore(ctx context.Context, idStr string) (ProductAdminResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return ProductAdminResponse{}, fmt.Errorf("invalid product id")
	}

	_, err = s.repo.Restore(ctx, id)
	if err != nil {
		return ProductAdminResponse{}, err
	}

	return s.GetByIDAdmin(ctx, idStr)
}

func (s *Service) mapToPublicResponse(rows []dbgen.ListProductsPublicRow) ([]ProductResponse, int64, error) {
	var total int64
	res := make([]ProductResponse, 0)
	for _, row := range rows {
		if total == 0 {
			total = row.TotalCount
		}
		priceFloat, _ := strconv.ParseFloat(row.Price, 64)
		res = append(res, ProductResponse{
			ID:           row.ID.String(),
			CategoryName: row.CategoryName,
			Name:         row.Name,
			Slug:         row.Slug,
			Price:        priceFloat,
		})
	}
	return res, total, nil
}

func (s *Service) mapToAdminResponse(rows []dbgen.ListProductsAdminRow) ([]ProductAdminResponse, int64, error) {
	var total int64
	res := make([]ProductAdminResponse, 0)
	for _, row := range rows {
		if total == 0 {
			total = row.TotalCount
		}
		priceFloat, _ := strconv.ParseFloat(row.Price, 64)
		res = append(res, ProductAdminResponse{
			ID:           row.ID.String(),
			CategoryName: row.CategoryName,
			Name:         row.Name,
			Slug:         row.Slug,
			Price:        priceFloat,
			Stock:        row.Stock,
			SKU:          row.Sku.String,
			IsActive:     row.IsActive.Bool,
			CreatedAt:    row.CreatedAt,
		})
	}
	return res, total, nil
}

## Controller

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

func NewHandler(s *Service) *Controller {
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

	data, total, err := h.service.ListPublic(c.Request.Context(), req)
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

	data, total, err := h.service.ListAdmin(
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

	res, err := h.service.Create(c.Request.Context(), req)
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
	res, err := h.service.GetByIDAdmin(c.Request.Context(), c.Param("id"))
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

	res, err := h.service.Update(c.Request.Context(), id, req)
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
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
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
	res, err := h.service.Restore(c.Request.Context(), c.Param("id"))
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
