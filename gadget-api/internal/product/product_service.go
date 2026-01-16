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
