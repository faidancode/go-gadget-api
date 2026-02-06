package brand

import (
	"context"
	"database/sql"
	"fmt"
	branderrors "go-gadget-api/internal/brand/errors"
	"go-gadget-api/internal/cloudinary"
	"go-gadget-api/internal/dbgen"
	"go-gadget-api/internal/pkg/apperror"
	"go-gadget-api/internal/pkg/constants"
	"mime/multipart"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type CloudinaryService interface {
	UploadImage(ctx context.Context, file multipart.File, filename string, folderName string) (string, error)
	DeleteImage(ctx context.Context, publicID string) error
}

//go:generate mockgen -source=brand_service.go -destination=../mock/brand/brand_service_mock.go -package=mock
type Service interface {
	Create(ctx context.Context, req CreateBrandRequest, file multipart.File, filename string) (BrandAdminResponse, error)
	ListPublic(ctx context.Context, page, limit int) ([]BrandPublicResponse, int64, error)
	ListAdmin(ctx context.Context, req ListBrandRequest) ([]BrandAdminResponse, int64, error)
	GetByID(ctx context.Context, id string) (BrandAdminResponse, error)
	Update(ctx context.Context, id string, req UpdateBrandRequest, file multipart.File, filename string) (BrandAdminResponse, error)
	Delete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) (BrandAdminResponse, error)
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

func (s *service) ListPublic(ctx context.Context, page, limit int) ([]BrandPublicResponse, int64, error) {
	offset := (page - 1) * limit
	rows, err := s.repo.ListPublic(ctx, int32(limit), int32(offset))
	if err != nil {
		return nil, 0, err
	}

	var total int64
	res := make([]BrandPublicResponse, 0)
	for _, row := range rows {
		if total == 0 {
			total = row.TotalCount
		}
		res = append(res, BrandPublicResponse{
			ID:       row.ID.String(),
			Name:     row.Name,
			Slug:     row.Slug,
			ImageUrl: row.ImageUrl.String,
		})
	}
	return res, total, nil
}

// internal/brand/service.go

func (s *service) ListAdmin(ctx context.Context, req ListBrandRequest) ([]BrandAdminResponse, int64, error) {
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
	params := dbgen.ListBrandsAdminParams{
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

func (s *service) GetByID(ctx context.Context, idStr string) (BrandAdminResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return BrandAdminResponse{}, branderrors.ErrInvalidUUID
	}

	brand, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return BrandAdminResponse{}, branderrors.ErrBrandNotFound
	}
	return mapToResponse(brand), err
}

func (s *service) Create(ctx context.Context, req CreateBrandRequest, file multipart.File, filename string) (BrandAdminResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return BrandAdminResponse{}, apperror.MapValidationError(err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return BrandAdminResponse{}, err
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)
	brand, err := qtx.Create(ctx, dbgen.CreateBrandParams{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: dbgen.NewNullString(req.Description),
		ImageUrl:    sql.NullString{},
	})
	if err != nil {
		return BrandAdminResponse{}, err
	}

	// 4. Upload image ke Cloudinary (jika ada file)
	var imageURL string
	if file != nil && filename != "" {
		// Gunakan Brand ID agar nama file unik
		uniqueFilename := fmt.Sprintf("brand-%s-%s", brand.ID.String(), filename)

		imageURL, err = s.cloudinaryRepo.UploadImage(ctx, file, uniqueFilename, constants.CloudinaryBrandFolder)
		if err != nil {
			// Jika upload gagal, transaksi di-rollback otomatis oleh defer tx.Rollback()
			return BrandAdminResponse{}, branderrors.ErrImageUploadFailed
		}

		// 5. Update brand dengan image URL yang didapat
		_, err = qtx.Update(ctx, dbgen.UpdateBrandParams{
			ID:          brand.ID,
			Name:        brand.Name,
			Slug:        brand.Slug,
			Description: brand.Description,
			ImageUrl:    dbgen.NewNullString(imageURL),
			IsActive:    brand.IsActive,
		})
		if err != nil {
			// Update gagal, hapus image yang sudah terlanjur diupload
			_ = s.cloudinaryRepo.DeleteImage(ctx, uniqueFilename)
			return BrandAdminResponse{}, branderrors.ErrImageDeleteFailed
		}
	}

	// 6. Commit transaction
	if err := tx.Commit(); err != nil {
		// Jika commit gagal, hapus image jika tadi berhasil diupload
		if imageURL != "" {
			uniqueFilename := fmt.Sprintf("brand-%s-%s", brand.ID.String(), filename)
			_ = s.cloudinaryRepo.DeleteImage(ctx, uniqueFilename)
		}
		return BrandAdminResponse{}, branderrors.ErrBrandFailed
	}

	return s.GetByID(ctx, brand.ID.String())
}

func (s *service) Update(
	ctx context.Context,
	idStr string,
	req UpdateBrandRequest,
	file multipart.File,
	filename string,
) (BrandAdminResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return BrandAdminResponse{}, apperror.MapValidationError(err)
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return BrandAdminResponse{}, err
	}

	// 1. Ambil data lama
	brand, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return BrandAdminResponse{}, err
	}

	var newImageURL sql.NullString

	// 2. Kalau upload image baru
	if file != nil && filename != "" {

		// 2a. Hapus image lama (jika ada)
		if brand.ImageUrl.Valid && brand.ImageUrl.String != "" {
			publicID, err := cloudinary.ExtractPublicID(
				brand.ImageUrl.String,
				constants.CloudinaryBrandFolder,
			)
			if err != nil {
				return BrandAdminResponse{}, branderrors.ErrInvalidImageURL
			}

			_ = s.cloudinaryRepo.DeleteImage(ctx, publicID)
			// ⚠️ sengaja tidak fatal → image lama boleh gagal hapus
		}

		// 2b. Upload image baru
		uniqueFilename := fmt.Sprintf("brand-%s-%s", brand.ID, filename)

		imageURL, err := s.cloudinaryRepo.UploadImage(
			ctx,
			file,
			uniqueFilename,
			constants.CloudinaryBrandFolder,
		)
		if err != nil {
			return BrandAdminResponse{}, branderrors.ErrImageUploadFailed
		}

		newImageURL = dbgen.NewNullString(imageURL)
	} else {
		newImageURL = brand.ImageUrl
	}

	// 3. Update DB
	_, err = s.repo.Update(ctx, dbgen.UpdateBrandParams{
		ID:          brand.ID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: dbgen.NewNullString(req.Description),
		ImageUrl:    newImageURL,
		IsActive:    brand.IsActive,
	})
	if err != nil {
		return BrandAdminResponse{}, branderrors.ErrBrandFailed
	}

	return s.GetByID(ctx, brand.ID.String())
}

func (s *service) Delete(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}

	// 1. ambil data brand dulu
	brand, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 2. delete image jika ada
	if brand.ImageUrl.Valid && brand.ImageUrl.String != "" {
		publicID, err := cloudinary.ExtractPublicID(brand.ImageUrl.String, constants.CloudinaryBrandFolder)
		if err != nil {
			return fmt.Errorf("failed to extract public id: %w", err)
		}

		if err := s.cloudinaryRepo.DeleteImage(ctx, publicID); err != nil {
			return err
		}
	}

	// 3. delete brand di database
	return s.repo.Delete(ctx, id)
}

func (s *service) Restore(ctx context.Context, idStr string) (BrandAdminResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return BrandAdminResponse{}, err
	}
	brand, err := s.repo.Restore(ctx, id)
	return mapToResponse(brand), err
}

func mapToResponse(brand dbgen.Brand) BrandAdminResponse {
	return BrandAdminResponse{
		ID:        brand.ID.String(),
		Name:      brand.Name,
		ImageUrl:  brand.ImageUrl.String,
		Slug:      brand.Slug,
		CreatedAt: brand.CreatedAt,
	}
}

func (s *service) mapAdminRowsToResponse(rows []dbgen.ListBrandsAdminRow) []BrandAdminResponse {
	res := make([]BrandAdminResponse, 0)
	for _, row := range rows {
		res = append(res, BrandAdminResponse{
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
