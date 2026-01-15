package category

import (
	"context"
	"gadget-api/internal/dbgen"
	"strings"

	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateCategoryRequest) (CategoryResponse, error) {
	slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	cat, err := s.repo.Create(ctx, dbgen.CreateCategoryParams{
		Name:        req.Name,
		Slug:        slug,
		Description: dbgen.NewNullString(req.Description),
		ImageUrl:    dbgen.NewNullString(req.ImageUrl),
	})
	return mapToResponse(cat), err
}

func (s *Service) GetAll(ctx context.Context, page, limit int) ([]CategoryResponse, int64, error) {
	offset := (page - 1) * limit
	rows, err := s.repo.List(ctx, int32(limit), int32(offset))
	if err != nil {
		return nil, 0, err
	}

	var total int64 = 0
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

func (s *Service) GetByID(ctx context.Context, idStr string) (CategoryResponse, error) {
	id, _ := uuid.Parse(idStr)
	cat, err := s.repo.GetByID(ctx, id)
	return mapToResponse(cat), err
}

func (s *Service) Update(ctx context.Context, idStr string, req CreateCategoryRequest) (CategoryResponse, error) {
	id, _ := uuid.Parse(idStr)
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

func (s *Service) Delete(ctx context.Context, idStr string) error {
	id, _ := uuid.Parse(idStr)
	return s.repo.Delete(ctx, id)
}

func (s *Service) Restore(ctx context.Context, idStr string) (CategoryResponse, error) {
	id, _ := uuid.Parse(idStr)
	cat, err := s.repo.Restore(ctx, id)
	return mapToResponse(cat), err
}

// Helper Mapper
func mapToResponse(cat dbgen.Category) CategoryResponse {
	return CategoryResponse{
		ID:        cat.ID.String(),
		Name:      cat.Name,
		Slug:      cat.Slug,
		CreatedAt: cat.CreatedAt,
	}
}
