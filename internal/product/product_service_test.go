package product_test

import (
	"context"
	"database/sql"
	"errors"
	"mime/multipart"
	"testing"
	"time"

	"go-gadget-api/internal/pkg/constants"
	"go-gadget-api/internal/product"
	"go-gadget-api/internal/shared/database/dbgen"

	categoryMock "go-gadget-api/internal/mock/category"
	cloudinaryMock "go-gadget-api/internal/mock/cloudinary"
	productMock "go-gadget-api/internal/mock/product"
	reviewMock "go-gadget-api/internal/mock/review"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

//
// ======================= HELPERS =======================
//

type serviceDeps struct {
	db         *sql.DB
	sqlMock    sqlmock.Sqlmock
	service    product.Service
	repo       *productMock.MockRepository
	catRepo    *categoryMock.MockRepository
	reviewRepo *reviewMock.MockRepository
	cloudinary *cloudinaryMock.MockService
}

func setupServiceTest(t *testing.T) *serviceDeps {
	t.Helper()

	ctrl := gomock.NewController(t)
	db, sqlMock, _ := sqlmock.New()

	repo := productMock.NewMockRepository(ctrl)
	catRepo := categoryMock.NewMockRepository(ctrl)
	reviewRepo := reviewMock.NewMockRepository(ctrl)
	cloudinary := cloudinaryMock.NewMockService(ctrl)

	svc := product.NewService(db, repo, catRepo, reviewRepo, cloudinary)

	return &serviceDeps{
		db:         db,
		sqlMock:    sqlMock,
		service:    svc,
		repo:       repo,
		catRepo:    catRepo,
		reviewRepo: reviewRepo,
		cloudinary: cloudinary,
	}
}

func expectTx(t *testing.T, mock sqlmock.Sqlmock, commit bool) {
	t.Helper()

	mock.ExpectBegin()
	if commit {
		mock.ExpectCommit()
	} else {
		mock.ExpectRollback()
	}
}

type mockFile struct {
	multipart.File
}

func (m *mockFile) Read(p []byte) (n int, err error) { return 0, nil }
func (m *mockFile) Close() error                     { return nil }

//
// ======================= CREATE =======================
//

func TestProductService_Create(t *testing.T) {
	deps := setupServiceTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	brandID := uuid.New()
	catID := uuid.New()
	productID := uuid.New()

	req := product.CreateProductRequest{
		BrandID:    brandID.String(),
		CategoryID: catID.String(),
		Name:       "iPhone 15",
		Price:      15000000,
		Stock:      10,
	}

	t.Run("positive - success with image upload", func(t *testing.T) {
		// PERBAIKAN: Gunakan interface matcher untuk file yang tidak nil
		// Jika Anda ingin benar-benar mensimulasikan file, Anda butuh dummy struct yang mengimplementasikan multipart.File

		expectTx(t, deps.sqlMock, true)

		deps.repo.EXPECT().WithTx(gomock.Any()).Return(deps.repo)
		deps.catRepo.EXPECT().GetByID(gomock.Any(), catID).Return(dbgen.Category{ID: catID}, nil)
		deps.repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, arg dbgen.CreateProductParams) (dbgen.Product, error) {
				assert.True(t, arg.BrandID.Valid)
				assert.Equal(t, brandID, arg.BrandID.UUID)
				return dbgen.Product{
					ID:         productID,
					CategoryID: catID,
					BrandID:    uuid.NullUUID{UUID: brandID, Valid: true},
				}, nil
			},
		)

		// UploadImage akan dipanggil karena kita akan passing 'not nil' value di pemanggilan service
		deps.cloudinary.EXPECT().
			UploadImage(gomock.Any(), gomock.Any(), gomock.Any(), constants.CloudinaryProductFolder).
			Return("https://img.jpg", nil)

		deps.repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(dbgen.Product{}, nil)

		deps.repo.EXPECT().
			GetByID(gomock.Any(), productID).
			Return(dbgen.GetProductByIDRow{
				ID:        productID,
				Name:      req.Name,
				ImageUrl:  sql.NullString{String: "https://img.jpg", Valid: true},
				CreatedAt: time.Now(),
			}, nil)

		// PERBAIKAN: Jangan kirim nil jika ekspektasi mock adalah dipanggil.
		// Kita bisa menggunakan mock implementasi multipart.File atau cast dummy pointer.
		fakeFile := &mockFile{}
		res, err := deps.service.Create(ctx, req, fakeFile, "img.jpg")

		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("negative - image upload failed should rollback", func(t *testing.T) {
		// Gunakan pointer kosong atau implementasi dummy agar tidak nil saat dipanggil
		fakeFile := &mockFile{}

		expectTx(t, deps.sqlMock, false)

		deps.repo.EXPECT().
			WithTx(gomock.Any()).
			Return(deps.repo)

		deps.catRepo.EXPECT().
			GetByID(gomock.Any(), gomock.Any()).
			Return(dbgen.Category{ID: catID}, nil)

		deps.repo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, arg dbgen.CreateProductParams) (dbgen.Product, error) {
				assert.True(t, arg.BrandID.Valid)
				assert.Equal(t, brandID, arg.BrandID.UUID)
				return dbgen.Product{
					ID:         productID,
					CategoryID: catID,
					BrandID:    uuid.NullUUID{UUID: brandID, Valid: true},
				}, nil
			})

		// PERBAIKAN DI SINI:
		// Gunakan gomock.Any() untuk argumen kedua (file)
		deps.cloudinary.EXPECT().
			UploadImage(gomock.Any(), gomock.Any(), gomock.Any(), constants.CloudinaryProductFolder).
			Return("", errors.New("upload failed"))

		// Pastikan argumen 'fakeFile' dikirim di sini
		_, err := deps.service.Create(ctx, req, fakeFile, "img.jpg")

		assert.Error(t, err)
	})
}

//
// ======================= UPDATE =======================
//

func TestProductService_Update(t *testing.T) {
	deps := setupServiceTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	id := uuid.New()

	existing := dbgen.GetProductByIDRow{
		ID:   id,
		Name: "Old Name",
		ImageUrl: sql.NullString{
			String: "https://old.jpg",
			Valid:  true,
		},
		BrandID: uuid.NullUUID{}, // no brand initially
	}

	req := product.UpdateProductRequest{
		Name:    "New Name",
		BrandID: uuid.New().String(),
	}

	t.Run("positive - update brand id", func(t *testing.T) {
		newBrandID, _ := uuid.Parse(req.BrandID)

		expectTx(t, deps.sqlMock, true)

		// Fetch existing product
		deps.repo.EXPECT().GetByID(ctx, id).Return(existing, nil)

		deps.repo.EXPECT().WithTx(gomock.Any()).Return(deps.repo).AnyTimes()

		// verify the brand id is set in params passed to Update
		deps.repo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, arg dbgen.UpdateProductParams) (dbgen.Product, error) {
				assert.True(t, arg.BrandID.Valid)
				assert.Equal(t, newBrandID, arg.BrandID.UUID)
				return dbgen.Product{}, nil
			},
		)

		// final fetch for response
		deps.repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.GetProductByIDRow{ID: id, Name: req.Name}, nil)

		_, err := deps.service.Update(ctx, id.String(), req, nil, "")
		assert.NoError(t, err)
	})

	t.Run("negative - product not found", func(t *testing.T) {
		deps.repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.GetProductByIDRow{}, errors.New("not found"))

		// Case produk tidak ada, file bisa nil karena tidak akan sampai ke proses upload
		_, err := deps.service.Update(ctx, id.String(), req, nil, "")

		assert.Error(t, err)
	})
}

//
// ======================= DELETE =======================
//

func TestProductService_Delete(t *testing.T) {
	deps := setupServiceTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	id := uuid.New()
	imgUrl := "https://res.cloudinary.com/demo/image/upload/sample.jpg"

	t.Run("positive - delete with image cleanup", func(t *testing.T) {
		// 1. Mock GetByID untuk ambil info image
		deps.repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.GetProductByIDRow{
				ID:       id,
				ImageUrl: sql.NullString{String: imgUrl, Valid: true},
			}, nil)

		// 2. Mock Delete DB
		deps.repo.EXPECT().Delete(ctx, id).Return(nil)

		// 3. Mock Cloudinary Cleanup
		deps.cloudinary.EXPECT().DeleteImage(ctx, imgUrl).Return(nil)

		err := deps.service.Delete(ctx, id.String())

		assert.NoError(t, err)
	})

	t.Run("negative - product not found", func(t *testing.T) {
		deps.repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.GetProductByIDRow{}, errors.New("not found"))

		err := deps.service.Delete(ctx, id.String())

		assert.Error(t, err)
	})

	t.Run("positive - delete without image", func(t *testing.T) {
		// Case jika produk tidak punya image
		deps.repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.GetProductByIDRow{
				ID:       id,
				ImageUrl: sql.NullString{Valid: false},
			}, nil)

		deps.repo.EXPECT().Delete(ctx, id).Return(nil)
		// Cloudinary tidak boleh dipanggil

		err := deps.service.Delete(ctx, id.String())
		assert.NoError(t, err)
	})
}

//
// ======================= LIST PUBLIC =======================
//

func TestProductService_ListPublic(t *testing.T) {
	deps := setupServiceTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	req := product.ListPublicRequest{
		Page:      1,
		Limit:     10,
		BrandSlug: "apple",
	}

	t.Run("positive - success list public", func(t *testing.T) {
		// Mock data return
		rows := []dbgen.ListProductsPublicRow{
			{
				ID:           uuid.New(),
				Name:         "Product 1",
				Price:        "100.00",
				TotalCount:   1,
				CategoryName: "Cat 1",
			},
		}

		deps.repo.EXPECT().
			ListPublic(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, params dbgen.ListProductsPublicParams) ([]dbgen.ListProductsPublicRow, error) {
				assert.Equal(t, sql.NullString{String: "apple", Valid: true}, params.BrandSlug)
				assert.Equal(t, "999999999.00", params.MaxPrice)
				return rows, nil
			})

		res, total, err := deps.service.ListPublic(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, res, 1)
	})
}

//
// ======================= LIST ADMIN =======================
//

func TestProductService_ListAdmin(t *testing.T) {
	deps := setupServiceTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	req := product.ListProductAdminRequest{
		Page:    1,
		Limit:   5,
		SortBy:  "name",
		SortDir: "asc",
	}

	t.Run("positive - success list admin", func(t *testing.T) {
		// 1. Persiapan Data (Setup)
		productID := uuid.New()
		rows := []dbgen.ListProductsAdminRow{
			{
				ID:         productID,
				Name:       "Admin Product",
				Price:      "500.00",
				TotalCount: 1,
				IsActive:   sql.NullBool{Bool: true, Valid: true},
			},
		}

		// 2. Ekspektasi Params (Harus identik dengan hasil mapping di Service)
		// Field yang tidak diisi di Service (IsActive, CategoryID) akan menjadi Zero Value (Valid: false)
		expectedParams := dbgen.ListProductsAdminParams{
			Limit:   int32(req.Limit),
			Offset:  0,
			Search:  sql.NullString{String: "", Valid: false},
			SortCol: "name",
			SortDir: "asc",
		}

		// 3. Mock Expectation
		deps.repo.EXPECT().
			ListAdmin(gomock.Any(), expectedParams).
			Return(rows, nil).
			Times(1)

		// 4. Execution
		res, total, err := deps.service.ListAdmin(ctx, req)
		t.Logf("Response: %+v, Total: %d, Error: %v", res, total, err)

		// 5. Assertion
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, res, 1)
		assert.Equal(t, "Admin Product", res[0].Name)
		assert.Equal(t, productID.String(), res[0].ID)
	})

	t.Run("positive - list admin with defaults", func(t *testing.T) {
		// Test jika request kosong (safety defaults)
		emptyReq := product.ListProductAdminRequest{}

		deps.repo.EXPECT().
			ListAdmin(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, p dbgen.ListProductsAdminParams) ([]dbgen.ListProductsAdminRow, error) {
				// Pastikan default nilai yang diset service benar
				assert.Equal(t, int32(10), p.Limit)
				assert.Equal(t, "created_at", p.SortCol)
				assert.Equal(t, "desc", p.SortDir)
				return []dbgen.ListProductsAdminRow{}, nil
			})

		_, _, err := deps.service.ListAdmin(ctx, emptyReq)
		assert.NoError(t, err)
	})
}

//
// ======================= GET BY SLUG =======================
//

func TestProductService_GetBySlug(t *testing.T) {
	deps := setupServiceTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	id := uuid.New()
	slug := "iphone-15-abcde"

	t.Run("success", func(t *testing.T) {
		deps.repo.EXPECT().GetBySlug(ctx, slug).Return(dbgen.GetProductBySlugRow{
			ID: id, Name: "iPhone 15", Slug: slug, Price: "1500.00",
		}, nil)

		deps.reviewRepo.EXPECT().GetByProductID(ctx, id, int32(5), int32(0)).Return(nil, nil)
		deps.reviewRepo.EXPECT().GetAverageRating(ctx, id).Return(4.5, nil)
		deps.reviewRepo.EXPECT().CountByProductID(ctx, id).Return(int64(10), nil)

		res, err := deps.service.GetBySlug(ctx, slug)
		assert.NoError(t, err)
		assert.Equal(t, slug, res.Slug)
		assert.Equal(t, 4.5, res.AverageRating)
	})
}

//
// ======================= GET BY ID =======================
//

func TestProductService_GetByID(t *testing.T) {
	deps := setupServiceTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	id := uuid.New()

	t.Run("positive - success get by id", func(t *testing.T) {
		// Mock Repository GetByID
		deps.repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.GetProductByIDRow{
				ID:           id,
				CategoryName: "Electronics",
				Name:         "iPhone 15",
				Slug:         "iphone-15",
				Price:        "15000000.00",
				Stock:        10,
				IsActive:     sql.NullBool{Bool: true, Valid: true},
				CreatedAt:    time.Now(),
			}, nil)

		res, err := deps.service.GetByID(ctx, id.String())

		assert.NoError(t, err)
		assert.Equal(t, id.String(), res.ID)
		assert.Equal(t, float64(15000000), res.Price)
	})

	t.Run("negative - invalid uuid string", func(t *testing.T) {
		_, err := deps.service.GetByID(ctx, "invalid-uuid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid product id")
	})
}

//
// ======================= RESTORE =======================
//

func TestProductService_Restore(t *testing.T) {
	deps := setupServiceTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		deps.repo.EXPECT().Restore(ctx, id).Return(dbgen.Product{ID: id}, nil)

		// Restore memanggil GetByID di akhir
		deps.repo.EXPECT().GetByID(ctx, id).Return(dbgen.GetProductByIDRow{
			ID: id, Name: "Restored Product", Price: "100.00",
		}, nil)

		res, err := deps.service.Restore(ctx, id.String())
		assert.NoError(t, err)
		assert.Equal(t, id.String(), res.ID)
	})
}
