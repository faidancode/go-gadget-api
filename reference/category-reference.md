## Service
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


## Controller
package category

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

func (ctrl *Controller) GetAll(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	data, total, err := ctrl.service.GetAll(c.Request.Context(), page, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_ERROR", "Gagal mengambil kategori", err.Error())
		return
	}

	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	response.Success(c, http.StatusOK, data, &response.PaginationMeta{
		Total:      total,
		TotalPages: totalPages,
		Page:       page,
		PageSize:   limit,
	})
}

func (ctrl *Controller) GetByID(c *gin.Context) {
	res, err := ctrl.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(
			c,
			http.StatusNotFound,
			"NOT_FOUND",
			"Category not found",
			nil,
		)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

func (ctrl *Controller) Create(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Invalid input",
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
			"Failed to create category",
			err.Error(),
		)
		return
	}

	response.Success(c, http.StatusCreated, res, nil)
}

func (ctrl *Controller) Update(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Invalid input",
			err.Error(),
		)
		return
	}

	res, err := ctrl.service.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(
			c,
			http.StatusInternalServerError,
			"UPDATE_ERROR",
			"Failed to update category",
			err.Error(),
		)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

func (ctrl *Controller) Delete(c *gin.Context) {
	if err := ctrl.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		response.Error(
			c,
			http.StatusInternalServerError,
			"DELETE_ERROR",
			"Failed to delete category",
			err.Error(),
		)
		return
	}

	response.Success(c, http.StatusOK, nil, nil)
}

func (ctrl *Controller) Restore(c *gin.Context) {
	res, err := ctrl.service.Restore(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(
			c,
			http.StatusInternalServerError,
			"RESTORE_ERROR",
			"Failed to restore category",
			err.Error(),
		)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}
