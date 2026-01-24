package product

import (
	"context"
	"database/sql"
	"fmt"
	"gadget-api/internal/category"
	"gadget-api/internal/dbgen"
	"gadget-api/internal/pkg/constants"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ReviewRepository interface {
	GetByProductID(ctx context.Context, productID uuid.UUID, limit, offset int32) ([]dbgen.GetReviewsByProductIDRow, error)
	CountByProductID(ctx context.Context, productID uuid.UUID) (int64, error)
	GetAverageRating(ctx context.Context, productID uuid.UUID) (float64, error)
}

// ReviewRow represents review data from repository
type ReviewRow struct {
	ID        uuid.UUID
	UserName  string
	Rating    int32
	Comment   string
	CreatedAt time.Time
}

type CloudinaryService interface {
	UploadImage(ctx context.Context, file multipart.File, filename string, folderName string) (string, error)
	DeleteImage(ctx context.Context, publicID string) error
}

//go:generate mockgen -source=product_service.go -destination=../mock/product/product_service_mock.go -package=mock
type Service interface {
	ListPublic(ctx context.Context, req ListPublicRequest) ([]ProductPublicResponse, int64, error)
	ListAdmin(ctx context.Context, req ListProductAdminRequest) ([]ProductAdminResponse, int64, error)
	Create(ctx context.Context, req CreateProductRequest, file multipart.File, filename string) (ProductAdminResponse, error)
	Update(ctx context.Context, idStr string, req UpdateProductRequest, file multipart.File, filename string) (ProductAdminResponse, error)
	Delete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) (ProductAdminResponse, error)

	GetByID(ctx context.Context, id string) (ProductAdminResponse, error)
	GetBySlug(ctx context.Context, slug string) (ProductDetailResponse, error)
}

type service struct {
	db             *sql.DB
	repo           Repository
	categoryRepo   category.Repository
	reviewRepo     ReviewRepository
	cloudinaryRepo CloudinaryService
}

func NewService(db *sql.DB, repo Repository, categoryRepo category.Repository, reviewRepo ReviewRepository, cloudinaryRepo CloudinaryService) Service {
	return &service{
		db:             db,
		repo:           repo,
		categoryRepo:   categoryRepo,
		reviewRepo:     reviewRepo,
		cloudinaryRepo: cloudinaryRepo,
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

func (s *service) GetBySlug(ctx context.Context, slug string) (ProductDetailResponse, error) {
	// 1. Get product by slug
	product, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		if err == sql.ErrNoRows {
			return ProductDetailResponse{}, ErrProductNotFound
		}
		return ProductDetailResponse{}, ErrProductFailed
	}

	// 2. Get reviews (limit 5, newest first)
	reviews, err := s.reviewRepo.GetByProductID(ctx, product.ID, 5, 0)
	if err != nil && err != sql.ErrNoRows {
		reviews = nil // atau bisa nil
	}

	// 3. Get average rating
	avgRating, err := s.reviewRepo.GetAverageRating(ctx, product.ID)
	if err != nil && err != sql.ErrNoRows {
		avgRating = 0
	}

	// 4. Get total reviews count
	totalReviews, err := s.reviewRepo.CountByProductID(ctx, product.ID)
	if err != nil && err != sql.ErrNoRows {
		totalReviews = 0
	}

	// 5. Map to response
	return s.mapToDetailResponse(product, reviews, avgRating, totalReviews), nil
}

func (s *service) ListAdmin(
	ctx context.Context,
	req ListProductAdminRequest,
) ([]ProductAdminResponse, int64, error) {

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

	// normalize sort
	sortCol := req.SortBy
	if sortCol == "" {
		sortCol = "created_at"
	}

	sortDir := strings.ToLower(req.SortDir)
	if sortDir != "asc" {
		sortDir = "desc"
	}

	params := dbgen.ListProductsAdminParams{
		Limit:   int32(limit),
		Offset:  int32(offset),
		Search:  dbgen.NewNullString(req.Search),
		SortCol: sortCol,
		SortDir: sortDir,
	}

	if req.Category != "" {
		if uid, err := uuid.Parse(req.Category); err == nil {
			params.CategoryID = uuid.NullUUID{
				UUID:  uid,
				Valid: true,
			}
		}
	}

	rows, err := s.repo.ListAdmin(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	return s.mapToAdminResponse(rows)
}

func (s *service) Create(ctx context.Context, req CreateProductRequest, file multipart.File, filename string) (ProductAdminResponse, error) {
	// 1. Validate category
	catID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return ProductAdminResponse{}, ErrInvalidCategoryID
	}

	_, err = s.categoryRepo.GetByID(ctx, catID)
	if err != nil {
		return ProductAdminResponse{}, ErrCategoryNotFound
	}

	// 2. Generate slug
	slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-")) + "-" + uuid.New().String()[:5]
	priceStr := fmt.Sprintf("%.2f", req.Price)

	// 3. Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ProductAdminResponse{}, ErrProductFailed
	}
	defer tx.Rollback()

	// 4. Create product in DB (without image first)
	qtx := s.repo.WithTx(tx)
	product, err := qtx.Create(ctx, dbgen.CreateProductParams{
		CategoryID:  catID,
		Name:        req.Name,
		Slug:        slug,
		Description: dbgen.NewNullString(req.Description),
		Price:       priceStr,
		Stock:       req.Stock,
		Sku:         dbgen.NewNullString(req.SKU),
		ImageUrl:    sql.NullString{}, // Empty first
	})
	if err != nil {
		return ProductAdminResponse{}, ErrProductFailed
	}

	// 5. Upload image to Cloudinary (if provided)
	var imageURL string
	if file != nil && filename != "" {
		// Generate unique filename
		uniqueFilename := fmt.Sprintf("%s-%s", product.ID.String(), filename)
		imageURL, err = s.cloudinaryRepo.UploadImage(ctx, file, uniqueFilename, constants.CloudinaryProductFolder)
		if err != nil {
			// Upload failed, rollback transaction
			return ProductAdminResponse{}, fmt.Errorf("failed to upload image: %w", err)
		}

		// 6. Update product with image URL
		_, err = qtx.Update(ctx, dbgen.UpdateProductParams{
			ID:          product.ID,
			CategoryID:  product.CategoryID,
			Name:        product.Name,
			Description: product.Description,
			Price:       product.Price,
			Stock:       product.Stock,
			Sku:         product.Sku,
			ImageUrl:    dbgen.NewNullString(imageURL),
			IsActive:    product.IsActive,
		})
		if err != nil {
			// Update failed, should delete uploaded image
			_ = s.cloudinaryRepo.DeleteImage(ctx, uniqueFilename)
			return ProductAdminResponse{}, ErrProductFailed
		}
	}

	// 7. Commit transaction
	if err := tx.Commit(); err != nil {
		// Commit failed, delete uploaded image if exists
		if imageURL != "" {
			_ = s.cloudinaryRepo.DeleteImage(ctx, fmt.Sprintf("%s-%s", product.ID.String(), filename))
		}
		return ProductAdminResponse{}, ErrProductFailed
	}

	// 8. Return created product
	return s.GetByID(ctx, product.ID.String())
}

func (s *service) GetByID(ctx context.Context, idStr string) (ProductAdminResponse, error) {
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

// Update updates a product with optional image upload
func (s *service) Update(ctx context.Context, idStr string, req UpdateProductRequest, file multipart.File, filename string) (ProductAdminResponse, error) {
	// 1. Validate product ID
	id, err := uuid.Parse(idStr)
	if err != nil {
		return ProductAdminResponse{}, ErrInvalidProductID
	}

	// 2. Get existing product
	existingProduct, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ProductAdminResponse{}, ErrProductNotFound
	}

	// 3. Prepare update params
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

	// 4. Update fields if provided
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

	// 5. Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ProductAdminResponse{}, ErrProductFailed
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)

	// 6. Handle image upload if provided
	var newImageURL string
	var oldImageURL string
	if file != nil && filename != "" {
		// Store old image URL for cleanup
		oldImageURL = existingProduct.ImageUrl.String

		// Upload new image
		uniqueFilename := fmt.Sprintf("%s-%s", id.String(), filename)
		newImageURL, err = s.cloudinaryRepo.UploadImage(ctx, file, uniqueFilename, constants.CloudinaryProductFolder)
		if err != nil {
			return ProductAdminResponse{}, fmt.Errorf("failed to upload image: %w", err)
		}

		params.ImageUrl = dbgen.NewNullString(newImageURL)
	}

	// 7. Update product in DB
	_, err = qtx.Update(ctx, params)
	if err != nil {
		// Update failed, delete new uploaded image if exists
		if newImageURL != "" {
			_ = s.cloudinaryRepo.DeleteImage(ctx, fmt.Sprintf("%s-%s", id.String(), filename))
		}
		return ProductAdminResponse{}, ErrProductFailed
	}

	// 8. Commit transaction
	if err := tx.Commit(); err != nil {
		// Commit failed, cleanup new image
		if newImageURL != "" {
			_ = s.cloudinaryRepo.DeleteImage(ctx, fmt.Sprintf("%s-%s", id.String(), filename))
		}
		return ProductAdminResponse{}, ErrProductFailed
	}

	// 9. Delete old image from Cloudinary (after successful update)
	if oldImageURL != "" && newImageURL != "" {
		// Extract public ID from old URL and delete
		// This is fire-and-forget, we don't fail if deletion fails
		_ = s.cloudinaryRepo.DeleteImage(ctx, oldImageURL)
	}

	// 10. Return updated product
	return s.GetByID(ctx, idStr)
}

func (s *service) Delete(ctx context.Context, idStr string) error {
	// 1. Validate ID
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid product id")
	}

	// 2. Get existing product to get image URL
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 3. Delete from database
	err = s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	// 4. Cleanup Cloudinary if image exists
	if product.ImageUrl.Valid && product.ImageUrl.String != "" {
		// Fire and forget: kita tidak ingin gagalkan delete DB hanya karena cleanup image gagal
		_ = s.cloudinaryRepo.DeleteImage(ctx, product.ImageUrl.String)
	}

	return nil
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

	return s.GetByID(ctx, idStr)
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

func (s *service) mapToDetailResponse(
	product dbgen.GetProductBySlugRow,
	reviews []dbgen.GetReviewsByProductIDRow,
	avgRating float64,
	totalReviews int64,
) ProductDetailResponse {
	price, _ := strconv.ParseFloat(product.Price, 64)

	var reviewSummaries []ReviewSummary
	for _, r := range reviews {
		reviewSummaries = append(reviewSummaries, ReviewSummary{
			ID:        r.ID.String(),
			UserName:  r.UserName,
			Rating:    r.Rating,
			Comment:   r.Comment,
			CreatedAt: r.CreatedAt,
		})
	}

	return ProductDetailResponse{
		ID:            product.ID.String(),
		Name:          product.Name,
		Slug:          product.Slug,
		Description:   product.Description.String, // Handle sql.NullString
		Price:         price,
		Stock:         product.Stock,
		ImageURL:      product.ImageUrl.String, // Handle sql.NullString
		SKU:           product.Sku.String,
		CategoryID:    product.CategoryID.String(),
		Reviews:       reviewSummaries,
		AverageRating: avgRating,
		TotalReviews:  totalReviews,
		CreatedAt:     product.CreatedAt,
		UpdatedAt:     product.UpdatedAt,
	}
}

// Helper function to calculate average rating (if needed in other methods)
func calculateAverageRating(reviews []ReviewRow) float64 {
	if len(reviews) == 0 {
		return 0
	}

	var sum int32
	for _, r := range reviews {
		sum += r.Rating
	}

	return float64(sum) / float64(len(reviews))
}
