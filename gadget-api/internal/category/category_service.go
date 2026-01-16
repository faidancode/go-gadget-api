package category

import (
	"context"
	"gadget-api/internal/dbgen"
	"strings"

	"github.com/google/uuid"
)

//go:generate mockgen -source=category_service.go -destination=mock/category_service_mock.go -package=mock
type Service interface {
	Create(ctx context.Context, req CreateCategoryRequest) (CategoryResponse, error)
	GetAll(ctx context.Context, page, limit int) ([]CategoryResponse, int64, error)
	GetByID(ctx context.Context, id string) (CategoryResponse, error)
	Update(ctx context.Context, id string, req CreateCategoryRequest) (CategoryResponse, error)
	Delete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) (CategoryResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req CreateCategoryRequest) (CategoryResponse, error) {
	slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	cat, err := s.repo.Create(ctx, dbgen.CreateCategoryParams{
		Name:        req.Name,
		Slug:        slug,
		Description: dbgen.NewNullString(req.Description),
		ImageUrl:    dbgen.NewNullString(req.ImageUrl),
	})
	return mapToResponse(cat), err
}

func (s *service) GetAll(ctx context.Context, page, limit int) ([]CategoryResponse, int64, error) {
	offset := (page - 1) * limit
	rows, err := s.repo.List(ctx, int32(limit), int32(offset))
	if err != nil {
		return nil, 0, err
	}

	var total int64
	res := make([]CategoryResponse, 0)
	for _, row := range rows {
		if total == 0 {
			total = row.TotalCount
		}
		res = append(res, CategoryResponse{
			ID:   row.ID.String(),
			Name: row.Name,
			Slug: row.Slug,
		})
	}
	return res, total, nil
}

func (s *service) GetByID(ctx context.Context, idStr string) (CategoryResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return CategoryResponse{}, err
	}
	cat, err := s.repo.GetByID(ctx, id)
	return mapToResponse(cat), err
}

func (s *service) Update(ctx context.Context, idStr string, req CreateCategoryRequest) (CategoryResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return CategoryResponse{}, err
	}

	slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	cat, err := s.repo.Update(ctx, dbgen.UpdateCategoryParams{
		ID:          id,
		Name:        req.Name,
		Slug:        slug,
		Description: dbgen.NewNullString(req.Description),
		ImageUrl:    dbgen.NewNullString(req.ImageUrl),
	})
	return mapToResponse(cat), err
}

func (s *service) Delete(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *service) Restore(ctx context.Context, idStr string) (CategoryResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return CategoryResponse{}, err
	}
	cat, err := s.repo.Restore(ctx, id)
	return mapToResponse(cat), err
}

func mapToResponse(cat dbgen.Category) CategoryResponse {
	return CategoryResponse{
		ID:        cat.ID.String(),
		Name:      cat.Name,
		Slug:      cat.Slug,
		CreatedAt: cat.CreatedAt,
	}
}
