package category

import (
	"context"
	"database/sql"
	"fmt"
	categoryerrors "gadget-api/internal/category/errors"
	"gadget-api/internal/cloudinary"
	"gadget-api/internal/dbgen"
	"gadget-api/internal/pkg/apperror"
	"gadget-api/internal/pkg/constants"
	"mime/multipart"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type CloudinaryService interface {
	UploadImage(ctx context.Context, file multipart.File, filename string, folderName string) (string, error)
	DeleteImage(ctx context.Context, publicID string) error
}

//go:generate mockgen -source=category_service.go -destination=../mock/category/category_service_mock.go -package=mock
type Service interface {
	Create(ctx context.Context, req CreateCategoryRequest, file multipart.File, filename string) (CategoryAdminResponse, error)
	ListPublic(ctx context.Context, page, limit int) ([]CategoryPublicResponse, int64, error)
	ListAdmin(ctx context.Context, req ListCategoryRequest) ([]CategoryAdminResponse, int64, error)
	GetByID(ctx context.Context, id string) (CategoryAdminResponse, error)
	Update(ctx context.Context, id string, req UpdateCategoryRequest, file multipart.File, filename string) (CategoryAdminResponse, error)
	Delete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) (CategoryAdminResponse, error)
}

type service struct {
	db             *sql.DB
	repo           Repository
	cloudinaryRepo CloudinaryService
	validate       *validator.Validate
}

func NewService(db *sql.DB, repo Repository, cloudinaryRepo CloudinaryService) Service {
	return &service{
		db:             db,
		repo:           repo,
		cloudinaryRepo: cloudinaryRepo,
		validate:       validator.New(),
	}
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
			ID:       row.ID.String(),
			Name:     row.Name,
			Slug:     row.Slug,
			ImageUrl: row.ImageUrl.String,
		})
	}
	return res, total, nil
}

// internal/category/service.go

func (s *service) ListAdmin(ctx context.Context, req ListCategoryRequest) ([]CategoryAdminResponse, int64, error) {
	// 1. Pagination Safety
	limit := req.Limit
	if limit < 1 {
		limit = 10
	}
	offset := (req.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	// 2. Default Sorting
	sortCol := "created_at"
	sortDir := "desc"

	// 3. Parsing Format "name:asc"
	if req.Sort != "" {
		parts := strings.Split(req.Sort, ":")
		if len(parts) == 2 {
			sortCol = strings.ToLower(parts[0])
			sortDir = strings.ToLower(parts[1])
		}
	}

	// 4. Panggil SQLC
	params := dbgen.ListCategoriesAdminParams{
		Limit:   int32(limit),
		Offset:  int32(offset),
		Search:  dbgen.NewNullString(req.Search),
		SortCol: sortCol,
		SortDir: sortDir,
	}

	rows, err := s.repo.ListAdmin(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	// 5. Handle Total Count
	var total int64 = 0
	if len(rows) > 0 {
		total = rows[0].TotalCount
	}

	return s.mapAdminRowsToResponse(rows), total, nil
}

func (s *service) GetByID(ctx context.Context, idStr string) (CategoryAdminResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return CategoryAdminResponse{}, categoryerrors.ErrInvalidUUID
	}
	category, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return CategoryAdminResponse{}, categoryerrors.ErrCategoryNotFound
	}
	return mapToResponse(category), err
}

func (s *service) Create(ctx context.Context, req CreateCategoryRequest, file multipart.File, filename string) (CategoryAdminResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return CategoryAdminResponse{}, apperror.MapValidationError(err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CategoryAdminResponse{}, err
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)
	category, err := qtx.Create(ctx, dbgen.CreateCategoryParams{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: dbgen.NewNullString(req.Description),
		ImageUrl:    sql.NullString{},
	})
	if err != nil {
		return CategoryAdminResponse{}, categoryerrors.ErrCategoryFailed
	}

	// 4. Upload image ke Cloudinary (jika ada file)
	var imageURL string
	if file != nil && filename != "" {
		// Gunakan Category ID agar nama file unik
		uniqueFilename := fmt.Sprintf("category-%s-%s", category.ID.String(), filename)

		imageURL, err = s.cloudinaryRepo.UploadImage(ctx, file, uniqueFilename, constants.CloudinaryCategoryFolder)
		if err != nil {
			// Jika upload gagal, transaksi di-rollback otomatis oleh defer tx.Rollback()
			return CategoryAdminResponse{}, categoryerrors.ErrImageUploadFailed
		}

		// 5. Update category dengan image URL yang didapat
		_, err = qtx.Update(ctx, dbgen.UpdateCategoryParams{
			ID:          category.ID,
			Name:        category.Name,
			Slug:        category.Slug,
			Description: category.Description,
			ImageUrl:    dbgen.NewNullString(imageURL),
			IsActive:    category.IsActive,
		})
		if err != nil {
			// Update gagal, hapus image yang sudah terlanjur diupload
			_ = s.cloudinaryRepo.DeleteImage(ctx, uniqueFilename)
			return CategoryAdminResponse{}, categoryerrors.ErrImageUploadFailed
		}
	}

	// 6. Commit transaction
	if err := tx.Commit(); err != nil {
		// Jika commit gagal, hapus image jika tadi berhasil diupload
		if imageURL != "" {
			uniqueFilename := fmt.Sprintf("category-%s-%s", category.ID.String(), filename)
			_ = s.cloudinaryRepo.DeleteImage(ctx, uniqueFilename)
		}
		return CategoryAdminResponse{}, categoryerrors.ErrImageDeleteFailed
	}

	return s.GetByID(ctx, category.ID.String())
}

func (s *service) Update(
	ctx context.Context,
	idStr string,
	req UpdateCategoryRequest,
	file multipart.File,
	filename string,
) (CategoryAdminResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return CategoryAdminResponse{}, apperror.MapValidationError(err)
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return CategoryAdminResponse{}, categoryerrors.ErrInvalidUUID
	}

	// 1. Ambil data lama
	category, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return CategoryAdminResponse{}, categoryerrors.ErrCategoryNotFound
	}

	var newImageURL sql.NullString

	// 2. Kalau upload image baru
	if file != nil && filename != "" {

		// 2a. Hapus image lama (jika ada)
		if category.ImageUrl.Valid && category.ImageUrl.String != "" {
			publicID, err := cloudinary.ExtractPublicID(
				category.ImageUrl.String,
				constants.CloudinaryCategoryFolder,
			)
			if err != nil {
				return CategoryAdminResponse{}, categoryerrors.ErrInvalidImageURL
			}

			_ = s.cloudinaryRepo.DeleteImage(ctx, publicID)
			// ⚠️ sengaja tidak fatal → image lama boleh gagal hapus
		}

		// 2b. Upload image baru
		uniqueFilename := fmt.Sprintf("category-%s-%s", category.ID, filename)

		imageURL, err := s.cloudinaryRepo.UploadImage(
			ctx,
			file,
			uniqueFilename,
			constants.CloudinaryCategoryFolder,
		)
		if err != nil {
			return CategoryAdminResponse{}, categoryerrors.ErrImageUploadFailed
		}

		newImageURL = dbgen.NewNullString(imageURL)
	} else {
		newImageURL = category.ImageUrl
	}

	// 3. Update DB
	_, err = s.repo.Update(ctx, dbgen.UpdateCategoryParams{
		ID:          category.ID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: dbgen.NewNullString(req.Description),
		ImageUrl:    newImageURL,
		IsActive:    category.IsActive,
	})
	if err != nil {
		return CategoryAdminResponse{}, err
	}

	return s.GetByID(ctx, category.ID.String())
}

func (s *service) Delete(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}

	// 1. ambil data category dulu
	category, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 2. delete image jika ada
	if category.ImageUrl.Valid && category.ImageUrl.String != "" {
		publicID, err := cloudinary.ExtractPublicID(category.ImageUrl.String, constants.CloudinaryCategoryFolder)
		if err != nil {
			return categoryerrors.ErrInvalidImageURL
		}

		if err := s.cloudinaryRepo.DeleteImage(ctx, publicID); err != nil {
			return err
		}
	}

	// 3. delete category di database
	return s.repo.Delete(ctx, id)
}

func (s *service) Restore(ctx context.Context, idStr string) (CategoryAdminResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return CategoryAdminResponse{}, err
	}
	category, err := s.repo.Restore(ctx, id)
	return mapToResponse(category), err
}

func mapToResponse(category dbgen.Category) CategoryAdminResponse {
	return CategoryAdminResponse{
		ID:        category.ID.String(),
		Name:      category.Name,
		ImageUrl:  category.ImageUrl.String,
		Slug:      category.Slug,
		CreatedAt: category.CreatedAt,
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
