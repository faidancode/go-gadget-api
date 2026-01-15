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

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetAll(ctx context.Context, page, limit int, search, categoryID string) ([]ProductResponse, int64, error) {
	offset := (page - 1) * limit

	params := dbgen.ListProductsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	if search != "" {
		params.Search = sql.NullString{String: search, Valid: true}
	}

	if categoryID != "" {
		uid, _ := uuid.Parse(categoryID)
		params.CategoryID = uuid.NullUUID{UUID: uid, Valid: true}
	}

	rows, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	var total int64
	res := make([]ProductResponse, 0)
	for _, row := range rows {
		if total == 0 {
			total = row.TotalCount
		}
		priceFloat, _ := strconv.ParseFloat(row.Price, 64)
		res = append(res, ProductResponse{
			ID:           row.ID.String(),
			Name:         row.Name,
			Slug:         row.Slug,
			Price:        priceFloat,
			Stock:        row.Stock,
			CategoryName: row.CategoryName,
		})
	}
	return res, total, nil
}

func (s *Service) Create(ctx context.Context, req CreateProductRequest) (ProductResponse, error) {
	catID, _ := uuid.Parse(req.CategoryID)
	_, err := s.catRepo.GetByID(ctx, catID)
	if err != nil {
		return ProductResponse{}, fmt.Errorf("category not found")
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

	return ProductResponse{ID: p.ID.String(), Name: p.Name, Slug: p.Slug}, err
}

func (s *Service) GetByID(ctx context.Context, idStr string) (ProductResponse, error) {
	id, _ := uuid.Parse(idStr)
	p, err := s.repo.GetByID(ctx, id)
	priceFloat, _ := strconv.ParseFloat(p.Price, 64)
	return ProductResponse{
		ID:           p.ID.String(),
		Name:         p.Name,
		Price:        priceFloat,
		CategoryName: p.CategoryName,
	}, err
}

func (s *Service) Delete(ctx context.Context, idStr string) error {
	id, _ := uuid.Parse(idStr)
	return s.repo.Delete(ctx, id)
}

func (s *Service) Restore(ctx context.Context, idStr string) (ProductResponse, error) {
	id, _ := uuid.Parse(idStr)
	p, err := s.repo.Restore(ctx, id)
	return ProductResponse{ID: p.ID.String(), Name: p.Name}, err
}
