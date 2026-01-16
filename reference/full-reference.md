## Query

-- name: ListProductsPublic :many
SELECT p.*, c.name as category_name, count(*) OVER() AS total_count
FROM products p
JOIN categories c ON p.category_id = c.id
WHERE p.deleted_at IS NULL 
  AND p.is_active = true
  -- Gunakan sintaks ini agar sqlc membuat field CategoryID (NullUUID)
  AND (sqlc.narg('category_id')::uuid IS NULL OR p.category_id = sqlc.narg('category_id')::uuid)
  AND (sqlc.narg('search')::text IS NULL OR p.name ILIKE '%' || sqlc.narg('search')::text || '%')
  AND (p.price >= sqlc.arg('min_price')::decimal)
  AND (p.price <= sqlc.arg('max_price')::decimal)
ORDER BY 
    CASE WHEN sqlc.arg('sort_by')::text = 'newest' THEN p.created_at END DESC,
    CASE WHEN sqlc.arg('sort_by')::text = 'oldest' THEN p.created_at END ASC,
    CASE WHEN sqlc.arg('sort_by')::text = 'price_high' THEN p.price END DESC,
    CASE WHEN sqlc.arg('sort_by')::text = 'price_low' THEN p.price END ASC,
    p.created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListProductsAdmin :many
SELECT p.*, c.name as category_name, count(*) OVER() AS total_count
FROM products p
JOIN categories c ON p.category_id = c.id
WHERE (sqlc.narg('category_id')::uuid IS NULL OR p.category_id = sqlc.narg('category_id')::uuid)
  AND (sqlc.narg('search')::text IS NULL OR p.name ILIKE '%' || sqlc.narg('search')::text || '%' OR p.sku ILIKE '%' || sqlc.narg('search')::text || '%')
ORDER BY 
    CASE WHEN sqlc.arg('sort_col')::text = 'stock' THEN p.stock END ASC,
    CASE WHEN sqlc.arg('sort_col')::text = 'name' THEN p.name END ASC,
    p.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetProductByID :one
SELECT p.*, c.name as category_name 
FROM products p
JOIN categories c ON p.category_id = c.id
WHERE p.id = $1 AND p.deleted_at IS NULL LIMIT 1;

-- name: CreateProduct :one
INSERT INTO products (category_id, name, slug, description, price, stock, sku, image_url)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateProduct :one
UPDATE products
SET 
    category_id = $2,
    name = $3,
    description = $4,
    price = $5,
    stock = $6,
    sku = $7,
    image_url = $8,
    is_active = $9,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SoftDeleteProduct :exec
UPDATE products SET deleted_at = NOW() WHERE id = $1;

-- name: RestoreProduct :one
UPDATE products SET deleted_at = NULL WHERE id = $1 RETURNING *;

## Schema
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID NOT NULL REFERENCES categories(id),
    name VARCHAR(200) NOT NULL,
    slug VARCHAR(200) NOT NULL UNIQUE,
    description TEXT,
    price DECIMAL(12,2) NOT NULL DEFAULT 0,
    stock INTEGER NOT NULL DEFAULT 0,
    sku VARCHAR(100) UNIQUE,
    image_url TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

## Response
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
	Success bool                   `json:"success"`
	Data    interface{}            `json:"data"`
	Meta    *PaginationMeta        `json:"meta"`
	Error   map[string]interface{} `json:"error"`
}

func Success(c *gin.Context, status int, data interface{}, meta *PaginationMeta) {
	c.JSON(status, ApiEnvelope{
		Success: true,
		Data:    data,
		Meta:    meta,
		Error:   nil,
	})
}

func Error(c *gin.Context, status int, errorCode string, message string, details interface{}) {
	c.JSON(status, ApiEnvelope{
		Success: false,
		Data:    nil,
		Meta:    nil,
		Error: map[string]interface{}{
			"code":    errorCode,
			"message": message,
			"details": details,
		},
	})
}

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


## DTO
package product

import "time"

// ListPublicRequest digunakan untuk menampung query params dari Customer
type ListPublicRequest struct {
	Page       int
	Limit      int
	Search     string
	CategoryID string
	MinPrice   float64
	MaxPrice   float64
	SortBy     string
}

// ProductResponse adalah output ringkas untuk Customer
type ProductPublicResponse struct {
	ID           string  `json:"id"`
	CategoryName string  `json:"category_name"`
	Name         string  `json:"name"`
	Slug         string  `json:"slug"`
	Price        float64 `json:"price"`
}

// ProductAdminResponse adalah output detail untuk Dashboard Admin (Ini yang menyebabkan error)
type ProductAdminResponse struct {
	ID           string    `json:"id"`
	CategoryName string    `json:"category_name"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Price        float64   `json:"price"`
	Stock        int32     `json:"stock"`
	SKU          string    `json:"sku"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateProductRequest digunakan untuk input Admin saat membuat produk baru
type CreateProductRequest struct {
	CategoryID  string  `json:"category_id" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required"`
	Stock       int32   `json:"stock" binding:"required"`
	SKU         string  `json:"sku"`
	ImageUrl    string  `json:"image_url"`
}

type UpdateProductRequest struct {
	CategoryID  string  `json:"category_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int32   `json:"stock"`
	SKU         string  `json:"sku"`
	ImageUrl    string  `json:"image_url"`
	IsActive    *bool   `json:"is_active"` // Gunakan pointer agar bisa membedakan false (bool) dan nil (tidak dikirim)
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

type Service interface {
	ListPublic(ctx context.Context, req ListPublicRequest) ([]ProductPublicResponse, int64, error)
	ListAdmin(ctx context.Context, page, limit int, search, sortCol, categoryID string) ([]ProductAdminResponse, int64, error)
	Create(ctx context.Context, req CreateProductRequest) (ProductAdminResponse, error)
	GetByIDAdmin(ctx context.Context, id string) (ProductAdminResponse, error)
	Update(ctx context.Context, id string, req UpdateProductRequest) (ProductAdminResponse, error)
	Delete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) (ProductAdminResponse, error)
}

type service struct {
	repo    Repository
	catRepo category.Repository
}

func NewService(repo Repository, catRepo category.Repository) Service {
	return &service{
		repo:    repo,
		catRepo: catRepo,
	}
}

func (s *service) ListPublic(ctx context.Context, req ListPublicRequest) ([]ProductPublicResponse, int64, error) {
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

func (s *service) ListAdmin(ctx context.Context, page, limit int, search, sortCol, categoryID string) ([]ProductAdminResponse, int64, error) {
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

func (s *service) Create(ctx context.Context, req CreateProductRequest) (ProductAdminResponse, error) {
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

func (s *service) GetByIDAdmin(ctx context.Context, idStr string) (ProductAdminResponse, error) {
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

func (s *service) Update(ctx context.Context, idStr string, req UpdateProductRequest) (ProductAdminResponse, error) {
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

func (s *service) Delete(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid product id")
	}
	return s.repo.Delete(ctx, id)
}

func (s *service) Restore(ctx context.Context, idStr string) (ProductAdminResponse, error) {
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

func (s *service) mapToPublicResponse(rows []dbgen.ListProductsPublicRow) ([]ProductPublicResponse, int64, error) {
	var total int64
	res := make([]ProductPublicResponse, 0)
	for _, row := range rows {
		if total == 0 {
			total = row.TotalCount
		}
		priceFloat, _ := strconv.ParseFloat(row.Price, 64)
		res = append(res, ProductPublicResponse{
			ID:           row.ID.String(),
			CategoryName: row.CategoryName,
			Name:         row.Name,
			Slug:         row.Slug,
			Price:        priceFloat,
		})
	}
	return res, total, nil
}

func (s *service) mapToAdminResponse(rows []dbgen.ListProductsAdminRow) ([]ProductAdminResponse, int64, error) {
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


## Service Test
package product

import (
	"context"
	"errors"
	catMock "gadget-api/internal/category/mock"
	"gadget-api/internal/dbgen"
	"gadget-api/internal/product/mock"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ========================
// CREATE PRODUCT
// ========================
func TestService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	catRepo := catMock.NewMockRepository(ctrl)
	service := NewService(repo, catRepo)

	ctx := context.Background()
	catID := uuid.New()

	req := CreateProductRequest{
		CategoryID: catID.String(),
		Name:       "iPhone 15",
		Price:      15000000,
		Stock:      10,
	}

	t.Run("Success", func(t *testing.T) {
		repoID := uuid.New()

		catRepo.EXPECT().
			GetByID(ctx, catID).
			Return(dbgen.Category{ID: catID}, nil)

		repo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(dbgen.Product{ID: repoID}, nil)

		repo.EXPECT().
			GetByID(ctx, repoID).
			Return(dbgen.GetProductByIDRow{
				ID:           repoID,
				Name:         req.Name,
				Price:        "15000000.00",
				Stock:        10,
				IsActive:     dbgen.NewNullBool(true),
				CategoryName: "Phone",
				CreatedAt:    time.Now(),
			}, nil)

		res, err := service.Create(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, req.Name, res.Name)
	})

	t.Run("Invalid Category ID", func(t *testing.T) {
		_, err := service.Create(ctx, CreateProductRequest{
			CategoryID: "invalid-uuid",
		})

		assert.Error(t, err)
		assert.Equal(t, "invalid category id", err.Error())
	})

	t.Run("Category Not Found", func(t *testing.T) {
		catRepo.EXPECT().
			GetByID(ctx, catID).
			Return(dbgen.Category{}, errors.New("not found"))

		_, err := service.Create(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "category not found", err.Error())
	})
}

// ========================
// GET BY ID (ADMIN)
// ========================
func TestService_GetByIDAdmin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	service := NewService(repo, nil)

	ctx := context.Background()
	id := uuid.New()

	t.Run("Success", func(t *testing.T) {
		repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.GetProductByIDRow{
				ID:       id,
				Name:     "Macbook",
				Price:    "20000000.00",
				Stock:    5,
				IsActive: dbgen.NewNullBool(true),
			}, nil)

		res, err := service.GetByIDAdmin(ctx, id.String())

		assert.NoError(t, err)
		assert.Equal(t, "Macbook", res.Name)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		_, err := service.GetByIDAdmin(ctx, "invalid-id")

		assert.Error(t, err)
		assert.Equal(t, "invalid product id", err.Error())
	})

	t.Run("Not Found", func(t *testing.T) {
		repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.GetProductByIDRow{}, errors.New("not found"))

		_, err := service.GetByIDAdmin(ctx, id.String())
		assert.Error(t, err)
	})
}

// ========================
// UPDATE PRODUCT
// ========================
func TestService_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	service := NewService(repo, nil)

	ctx := context.Background()
	id := uuid.New()

	existing := dbgen.GetProductByIDRow{
		ID:    id,
		Name:  "Old Name",
		Price: "100.00",
		Stock: 5,
	}

	req := UpdateProductRequest{
		Name:  "New Name",
		Price: 200,
	}

	t.Run("Success", func(t *testing.T) {
		repo.EXPECT().
			GetByID(ctx, id).
			Return(existing, nil)

		repo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(dbgen.Product{}, nil)

		repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.GetProductByIDRow{
				ID:    id,
				Name:  req.Name,
				Price: "200.00",
				Stock: 5,
			}, nil)

		res, err := service.Update(ctx, id.String(), req)

		assert.NoError(t, err)
		assert.Equal(t, req.Name, res.Name)
	})

	t.Run("Product Not Found", func(t *testing.T) {
		repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.GetProductByIDRow{}, errors.New("not found"))

		_, err := service.Update(ctx, id.String(), req)

		assert.Error(t, err)
		assert.Equal(t, "product not found", err.Error())
	})
}

// ========================
// DELETE PRODUCT
// ========================
func TestService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	service := NewService(repo, nil)

	ctx := context.Background()
	id := uuid.New()

	t.Run("Success", func(t *testing.T) {
		repo.EXPECT().
			Delete(ctx, id).
			Return(nil)

		err := service.Delete(ctx, id.String())
		assert.NoError(t, err)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		err := service.Delete(ctx, "invalid-id")
		assert.Error(t, err)
	})
}

// ========================
// RESTORE PRODUCT
// ========================
func TestService_Restore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	service := NewService(repo, nil)

	ctx := context.Background()
	id := uuid.New()

	repo.EXPECT().
		Restore(ctx, id).
		Return(dbgen.Product{}, nil)

	repo.EXPECT().
		GetByID(ctx, id).
		Return(dbgen.GetProductByIDRow{
			ID:    id,
			Name:  "Restored",
			Price: "100.00",
		}, nil)

	res, err := service.Restore(ctx, id.String())

	assert.NoError(t, err)
	assert.Equal(t, "Restored", res.Name)
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


## Controller Test
package product

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"gadget-api/internal/pkg/response"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

/*
====================================================
FAKE SERVICE (UNTUK CONTROLLER TEST)
====================================================
*/
type fakeProductService struct {
	ListPublicFn func(ctx context.Context, req ListPublicRequest) ([]ProductPublicResponse, int64, error)
	ListAdminFn  func(ctx context.Context, page, limit int, search, sortCol, categoryID string) ([]ProductAdminResponse, int64, error)
	CreateFn     func(ctx context.Context, req CreateProductRequest) (ProductAdminResponse, error)
	GetByIDFn    func(ctx context.Context, id string) (ProductAdminResponse, error)
	UpdateFn     func(ctx context.Context, id string, req UpdateProductRequest) (ProductAdminResponse, error)
	DeleteFn     func(ctx context.Context, id string) error
	RestoreFn    func(ctx context.Context, id string) (ProductAdminResponse, error)
}

func (f *fakeProductService) ListPublic(ctx context.Context, req ListPublicRequest) ([]ProductPublicResponse, int64, error) {
	return f.ListPublicFn(ctx, req)
}
func (f *fakeProductService) ListAdmin(ctx context.Context, p, l int, s, c, cid string) ([]ProductAdminResponse, int64, error) {
	return f.ListAdminFn(ctx, p, l, s, c, cid)
}
func (f *fakeProductService) Create(ctx context.Context, r CreateProductRequest) (ProductAdminResponse, error) {
	return f.CreateFn(ctx, r)
}
func (f *fakeProductService) GetByIDAdmin(ctx context.Context, id string) (ProductAdminResponse, error) {
	return f.GetByIDFn(ctx, id)
}
func (f *fakeProductService) Update(ctx context.Context, id string, r UpdateProductRequest) (ProductAdminResponse, error) {
	return f.UpdateFn(ctx, id, r)
}
func (f *fakeProductService) Delete(ctx context.Context, id string) error {
	return f.DeleteFn(ctx, id)
}
func (f *fakeProductService) Restore(ctx context.Context, id string) (ProductAdminResponse, error) {
	return f.RestoreFn(ctx, id)
}

/*
====================================================
SETUP ROUTER
====================================================
*/
func setupTest() (*gin.Engine, *fakeProductService) {
	gin.SetMode(gin.TestMode)

	svc := &fakeProductService{}
	ctrl := NewController(svc)

	r := gin.New()
	r.GET("/products", ctrl.GetPublicList)
	r.GET("/products/admin", ctrl.GetAdminList)
	r.POST("/products", ctrl.Create)
	r.GET("/products/:id", ctrl.GetByID)
	r.PUT("/products/:id", ctrl.Update)
	r.DELETE("/products/:id", ctrl.Delete)
	r.POST("/products/:id/restore", ctrl.Restore)

	return r, svc
}

/*
====================================================
GET PUBLIC LIST
====================================================
*/
func TestGetPublicList(t *testing.T) {
	router, svc := setupTest()

	t.Run("Success", func(t *testing.T) {
		svc.ListPublicFn = func(ctx context.Context, req ListPublicRequest) ([]ProductPublicResponse, int64, error) {
			return []ProductPublicResponse{}, 10, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/products?page=1&limit=5", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var res response.ApiEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		assert.True(t, res.Success)
		assert.NotNil(t, res.Meta)
	})

	t.Run("Internal Error", func(t *testing.T) {
		svc.ListPublicFn = func(ctx context.Context, req ListPublicRequest) ([]ProductPublicResponse, int64, error) {
			return nil, 0, errors.New("db error")
		}

		req := httptest.NewRequest(http.MethodGet, "/products", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

/*
====================================================
GET ADMIN LIST
====================================================
*/
func TestGetAdminList(t *testing.T) {
	router, svc := setupTest()

	svc.ListAdminFn = func(ctx context.Context, page, limit int, search, sortCol, categoryID string) ([]ProductAdminResponse, int64, error) {
		return []ProductAdminResponse{}, 0, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/products/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

/*
====================================================
CREATE PRODUCT
====================================================
*/
func TestCreateProduct(t *testing.T) {
	router, svc := setupTest()

	payload := CreateProductRequest{
		Name:       "Macbook",
		Price:      20000000,
		Stock:      10,
		CategoryID: uuid.New().String(),
	}

	t.Run("Success", func(t *testing.T) {
		svc.CreateFn = func(ctx context.Context, req CreateProductRequest) (ProductAdminResponse, error) {
			return ProductAdminResponse{Name: req.Name}, nil
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Validation Error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		svc.CreateFn = func(ctx context.Context, req CreateProductRequest) (ProductAdminResponse, error) {
			return ProductAdminResponse{}, errors.New("create failed")
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

/*
====================================================
GET BY ID
====================================================
*/
func TestGetProductByID(t *testing.T) {
	router, svc := setupTest()
	id := uuid.New().String()

	t.Run("Found", func(t *testing.T) {
		svc.GetByIDFn = func(ctx context.Context, pid string) (ProductAdminResponse, error) {
			return ProductAdminResponse{ID: pid}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/products/"+id, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		svc.GetByIDFn = func(ctx context.Context, pid string) (ProductAdminResponse, error) {
			return ProductAdminResponse{}, errors.New("not found")
		}

		req := httptest.NewRequest(http.MethodGet, "/products/"+id, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

/*
====================================================
UPDATE PRODUCT
====================================================
*/
func TestUpdateProduct(t *testing.T) {
	router, svc := setupTest()
	id := uuid.New().String()

	payload := UpdateProductRequest{
		Name:  "Updated",
		Price: 9999,
	}

	t.Run("Success", func(t *testing.T) {
		svc.UpdateFn = func(ctx context.Context, pid string, req UpdateProductRequest) (ProductAdminResponse, error) {
			return ProductAdminResponse{Name: req.Name}, nil
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPut, "/products/"+id, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		svc.UpdateFn = func(ctx context.Context, pid string, req UpdateProductRequest) (ProductAdminResponse, error) {
			return ProductAdminResponse{}, errors.New("product not found")
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPut, "/products/"+id, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

/*
====================================================
DELETE PRODUCT
====================================================
*/
func TestDeleteProduct(t *testing.T) {
	router, svc := setupTest()
	id := uuid.New().String()

	svc.DeleteFn = func(ctx context.Context, pid string) error {
		return nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/products/"+id, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

/*
====================================================
RESTORE PRODUCT
====================================================
*/
func TestRestoreProduct(t *testing.T) {
	router, svc := setupTest()
	id := uuid.New().String()

	svc.RestoreFn = func(ctx context.Context, pid string) (ProductAdminResponse, error) {
		return ProductAdminResponse{ID: pid}, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/products/"+id+"/restore", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}


## Seeder
package product

import (
	"context"
	"fmt"
	"gadget-api/internal/dbgen"
	"log"

	"github.com/google/uuid"
)

func SeedProducts(repo Repository, categoryID string, name string, price float64) {
	ctx := context.Background()
	catUUID, _ := uuid.Parse(categoryID)

	_, err := repo.Create(ctx, dbgen.CreateProductParams{
		CategoryID: catUUID,
		Name:       name,
		Slug:       fmt.Sprintf("%s-%s", name, uuid.New().String()[:4]),
		Price:      fmt.Sprintf("%.2f", price), // Konversi float ke string untuk DECIMAL
		Stock:      10,
		Sku:        dbgen.NewNullString("SKU-" + name),
	})

	if err != nil {
		log.Printf("Gagal seed produk %s: %v", name, err)
	} else {
		fmt.Printf("Berhasil seed produk: %s\n", name)
	}
}

