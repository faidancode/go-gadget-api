package category

import (
	"context"
	"gadget-api/internal/dbgen"
	"strings"

	"github.com/google/uuid"
)

//go:generate mockgen -source=category_service.go -destination=../mock/category/category_service_mock.go -package=mock
type Service interface {
	Create(ctx context.Context, req CreateCategoryRequest) (CategoryAdminResponse, error)
	ListPublic(ctx context.Context, page, limit int) ([]CategoryPublicResponse, int64, error)
	ListAdmin(ctx context.Context, req ListCategoryRequest) ([]CategoryAdminResponse, int64, error)
	GetByID(ctx context.Context, id string) (CategoryAdminResponse, error)
	Update(ctx context.Context, id string, req CreateCategoryRequest) (CategoryAdminResponse, error)
	Delete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) (CategoryAdminResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req CreateCategoryRequest) (CategoryAdminResponse, error) {
	slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	cat, err := s.repo.Create(ctx, dbgen.CreateCategoryParams{
		Name:        req.Name,
		Slug:        slug,
		Description: dbgen.NewNullString(req.Description),
		ImageUrl:    dbgen.NewNullString(req.ImageUrl),
	})
	return mapToResponse(cat), err
}

func (s *service) ListPublic(ctx context.Context, page, limit int) ([]CategoryPublicResponse, int64, error) {
	offset := (page - 1) * limit
	rows, err := s.repo.ListPublic(ctx, int32(limit), int32(offset))
	if err != nil {
		return nil, 0, err
	}

	var total int64
	res := make([]CategoryPublicResponse, 0)
	for _, row := range rows {
		if total == 0 {
			total = row.TotalCount
		}
		res = append(res, CategoryPublicResponse{
			ID:   row.ID.String(),
			Name: row.Name,
			Slug: row.Slug,
		})
	}
	return res, total, nil
}

// internal/category/service.go

func (s *service) ListAdmin(ctx context.Context, req ListCategoryRequest) ([]CategoryAdminResponse, int64, error) {
	// default safety
	page := req.Page
	limit := req.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	offset := (page - 1) * limit
	sortCol := req.SortBy
	if sortCol == "" {
		sortCol = "created_at"
	}

	sortDir := strings.ToLower(req.SortDir)
	if sortDir != "asc" {
		sortDir = "desc"
	}

	params := dbgen.ListCategoriesAdminParams{
		Limit:   int32(limit),
		Offset:  int32(offset),
		Search:  dbgen.NewNullString(req.Search),
		SortCol: sortCol,
		SortDir: sortDir,
	}

	// 2. Call Repository
	rows, err := s.repo.ListAdmin(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	// 3. Handle Empty State & Total Count
	var total int64 = 0
	if len(rows) > 0 {
		total = rows[0].TotalCount
	}

	// 4. Map Rows to Admin Response
	return s.mapAdminRowsToResponse(rows), total, nil
}

func (s *service) GetByID(ctx context.Context, idStr string) (CategoryAdminResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return CategoryAdminResponse{}, err
	}
	cat, err := s.repo.GetByID(ctx, id)
	return mapToResponse(cat), err
}

func (s *service) Update(ctx context.Context, idStr string, req CreateCategoryRequest) (CategoryAdminResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return CategoryAdminResponse{}, err
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

func (s *service) Restore(ctx context.Context, idStr string) (CategoryAdminResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return CategoryAdminResponse{}, err
	}
	cat, err := s.repo.Restore(ctx, id)
	return mapToResponse(cat), err
}

func mapToResponse(cat dbgen.Category) CategoryAdminResponse {
	return CategoryAdminResponse{
		ID:        cat.ID.String(),
		Name:      cat.Name,
		Slug:      cat.Slug,
		CreatedAt: cat.CreatedAt,
	}
}

func (s *service) mapAdminRowsToResponse(rows []dbgen.ListCategoriesAdminRow) []CategoryAdminResponse {
	res := make([]CategoryAdminResponse, 0)
	for _, row := range rows {
		res = append(res, CategoryAdminResponse{
			ID:          row.ID.String(),
			Name:        row.Name,
			Slug:        row.Slug,
			Description: row.Description.String,
			ImageUrl:    row.ImageUrl.String,
			IsActive:    row.IsActive.Bool,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
			DeletedAt:   nil, // Bisa diisi row.DeletedAt jika tipenya cocok
		})
	}
	return res
}
