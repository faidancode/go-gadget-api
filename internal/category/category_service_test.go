package category_test

import (
	"context"
	"database/sql"
	"errors"
	"mime/multipart"
	"testing"
	"time"

	"go-gadget-api/internal/category"
	"go-gadget-api/internal/dbgen"
	"go-gadget-api/internal/pkg/constants"

	categoryMock "go-gadget-api/internal/mock/category"
	cloudinaryMock "go-gadget-api/internal/mock/cloudinary"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ======================= HELPERS =======================

type serviceDeps struct {
	db         *sql.DB
	sqlMock    sqlmock.Sqlmock
	service    category.Service
	repo       *categoryMock.MockRepository
	cloudinary *cloudinaryMock.MockService
}

func setupServiceTest(t *testing.T) *serviceDeps {
	t.Helper()

	ctrl := gomock.NewController(t)
	db, sqlMock, _ := sqlmock.New()

	repo := categoryMock.NewMockRepository(ctrl)
	cloudinary := cloudinaryMock.NewMockService(ctrl)

	// Sesuaikan dengan constructor Category Service Anda yang baru
	svc := category.NewService(db, repo, cloudinary)

	return &serviceDeps{
		db:         db,
		sqlMock:    sqlMock,
		service:    svc,
		repo:       repo,
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

func TestCategoryService_ListPublic(t *testing.T) {
	deps := setupServiceTest(t)
	defer deps.db.Close()

	ctx := context.Background()

	t.Run("success list public", func(t *testing.T) {
		categoryID1 := uuid.New()
		categoryID2 := uuid.New()

		// Mock data dari database (SQLC Rows)
		mockRows := []dbgen.ListCategoriesPublicRow{
			{ID: categoryID1, Name: "Samsung", Slug: "samsung", TotalCount: 2},
			{ID: categoryID2, Name: "Sony", Slug: "sony", TotalCount: 2},
		}

		deps.repo.EXPECT().
			ListPublic(ctx, int32(10), int32(0)). // limit 10, offset 0
			Return(mockRows, nil)

		res, total, err := deps.service.ListPublic(ctx, 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, res, 2)
		assert.Equal(t, "Samsung", res[0].Name)
	})

	t.Run("error list public", func(t *testing.T) {
		deps.repo.EXPECT().
			ListPublic(ctx, gomock.Any(), gomock.Any()).
			Return(nil, errors.New("db error"))

		res, total, err := deps.service.ListPublic(ctx, 1, 10)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, int64(0), total)
	})
}

func TestCategoryService_ListAdmin(t *testing.T) {
	deps := setupServiceTest(t)
	defer deps.db.Close()

	ctx := context.Background()

	t.Run("success list admin with search and sort", func(t *testing.T) {
		req := category.ListCategoryRequest{
			Page:   1,
			Limit:  5,
			Search: "Apple",
			Sort:   "name:asc",
		}

		mockRows := []dbgen.ListCategoriesAdminRow{
			{
				ID:         uuid.New(),
				Name:       "Apple",
				Slug:       "apple",
				TotalCount: 1,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
		}

		// Verifikasi parameter yang dikirim ke repository
		deps.repo.EXPECT().
			ListAdmin(ctx, dbgen.ListCategoriesAdminParams{
				Limit:   int32(5),
				Offset:  int32(0),
				Search:  dbgen.NewNullString("Apple"),
				SortCol: "name",
				SortDir: "asc",
			}).
			Return(mockRows, nil)

		res, total, err := deps.service.ListAdmin(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, res, 1)
		assert.Equal(t, "Apple", res[0].Name)
	})

	t.Run("success list admin default pagination and sort", func(t *testing.T) {
		req := category.ListCategoryRequest{} // Empty request

		deps.repo.EXPECT().
			ListAdmin(ctx, dbgen.ListCategoriesAdminParams{
				Limit:   int32(10), // Default limit
				Offset:  int32(0),
				Search:  dbgen.NewNullString(""),
				SortCol: "created_at", // Default sort
				SortDir: "desc",       // Default dir
			}).
			Return([]dbgen.ListCategoriesAdminRow{}, nil)

		_, total, err := deps.service.ListAdmin(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, int64(0), total)
	})

	t.Run("error list admin", func(t *testing.T) {
		deps.repo.EXPECT().
			ListAdmin(ctx, gomock.Any()).
			Return(nil, errors.New("query error"))

		_, _, err := deps.service.ListAdmin(ctx, category.ListCategoryRequest{})

		assert.Error(t, err)
	})
}

func TestCategoryService_Create(t *testing.T) {
	deps := setupServiceTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	categoryID := uuid.New()
	req := category.CreateCategoryRequest{
		Name:        "Apple",
		Description: "Premium Tech Category",
	}

	t.Run("positive - success with image upload", func(t *testing.T) {
		fakeFile := &mockFile{}
		filename := "logo.png"
		imgURL := "https://cloudinary.com/apple.png"

		expectTx(t, deps.sqlMock, true)
		deps.repo.EXPECT().WithTx(gomock.Any()).Return(deps.repo)

		// 1. Create awal
		deps.repo.EXPECT().Create(ctx, gomock.Any()).Return(dbgen.Category{ID: categoryID, Name: req.Name}, nil)

		// 2. Upload Image
		deps.cloudinary.EXPECT().UploadImage(ctx, fakeFile, gomock.Any(), constants.CloudinaryCategoryFolder).Return(imgURL, nil)

		// 3. Update Image URL
		deps.repo.EXPECT().Update(ctx, gomock.Any()).Return(dbgen.Category{ID: categoryID}, nil)

		// 4. Final Get
		deps.repo.EXPECT().GetByID(ctx, categoryID).Return(dbgen.Category{
			ID: categoryID, Name: req.Name, ImageUrl: dbgen.NewNullString(imgURL),
		}, nil)

		res, err := deps.service.Create(ctx, req, fakeFile, filename)
		assert.NoError(t, err)
		assert.Equal(t, imgURL, res.ImageUrl)
	})

	t.Run("negative - upload image failed (rollback)", func(t *testing.T) {
		fakeFile := &mockFile{}
		expectTx(t, deps.sqlMock, false) // Expect Rollback
		deps.repo.EXPECT().WithTx(gomock.Any()).Return(deps.repo)

		deps.repo.EXPECT().Create(ctx, gomock.Any()).Return(dbgen.Category{ID: categoryID}, nil)

		// Simulasi upload gagal
		deps.cloudinary.EXPECT().UploadImage(ctx, gomock.Any(), gomock.Any(), constants.CloudinaryCategoryFolder).
			Return("", errors.New("cloudinary error"))

		res, err := deps.service.Create(ctx, req, fakeFile, "logo.png")
		assert.Error(t, err)
		assert.Empty(t, res)
	})

	t.Run("negative - database create failed", func(t *testing.T) {
		expectTx(t, deps.sqlMock, false)
		deps.repo.EXPECT().WithTx(gomock.Any()).Return(deps.repo)

		deps.repo.EXPECT().Create(ctx, gomock.Any()).Return(dbgen.Category{}, errors.New("db error"))

		_, err := deps.service.Create(ctx, req, nil, "")
		assert.Error(t, err)
	})
}

func TestCategoryService_Update(t *testing.T) {
	deps := setupServiceTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	id := uuid.New()
	req := category.UpdateCategoryRequest{Name: "Updated Apple"}

	t.Run("positive - update name only (no image)", func(t *testing.T) {
		deps.repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.Category{
				ID:       id,
				Name:     "Apple",
				ImageUrl: sql.NullString{},
				IsActive: dbgen.NewNullBool(true),
			}, nil)

		deps.repo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(dbgen.Category{ID: id}, nil)

		deps.repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.Category{
				ID:   id,
				Name: req.Name,
			}, nil)

		res, err := deps.service.Update(ctx, id.String(), req, nil, "")
		assert.NoError(t, err)
		assert.Equal(t, req.Name, res.Name)
	})

	t.Run("positive - update with new image", func(t *testing.T) {
		fakeFile := &mockFile{}
		imgURL := "https://res.cloudinary.com/demo/image/upload/v1769161193/go-gadget/categorys/category-new.png"

		deps.repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.Category{
				ID: id,
				ImageUrl: dbgen.NewNullString(
					"https://res.cloudinary.com/demo/image/upload/v1769161193/go-gadget/categorys/category-old.png",
				),
				IsActive: dbgen.NewNullBool(true),
			}, nil)

		deps.cloudinary.EXPECT().
			DeleteImage(ctx, "go-gadget/categorys/category-old").
			Return(nil)

		deps.cloudinary.EXPECT().
			UploadImage(ctx, fakeFile, gomock.Any(), constants.CloudinaryCategoryFolder).
			Return(imgURL, nil)

		deps.repo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(dbgen.Category{ID: id}, nil)

		deps.repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.Category{
				ID:       id,
				Name:     req.Name,
				ImageUrl: dbgen.NewNullString(imgURL),
			}, nil)

		res, err := deps.service.Update(ctx, id.String(), req, fakeFile, "new.png")
		assert.NoError(t, err)
		assert.Equal(t, imgURL, res.ImageUrl)
	})

	t.Run("negative - invalid uuid format", func(t *testing.T) {
		_, err := deps.service.Update(ctx, "invalid-uuid", req, nil, "")
		assert.Error(t, err)
	})

	t.Run("negative - category not found", func(t *testing.T) {
		deps.repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.Category{}, sql.ErrNoRows)

		_, err := deps.service.Update(ctx, id.String(), req, nil, "")
		assert.Error(t, err)
	})
}

func TestCategoryService_Delete(t *testing.T) {
	deps := setupServiceTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	id := uuid.New()

	t.Run("success - no image", func(t *testing.T) {
		category := dbgen.Category{
			ID:       id,
			ImageUrl: sql.NullString{Valid: false},
		}

		// 1. mock GetByID
		deps.repo.EXPECT().
			GetByID(ctx, id).
			Return(category, nil)

		// 2. mock Delete
		deps.repo.EXPECT().
			Delete(ctx, id).
			Return(nil)

		err := deps.service.Delete(ctx, id.String())
		assert.NoError(t, err)
	})

	t.Run("success - with image", func(t *testing.T) {
		imageURL := "https://res.cloudinary.com/demo/image/upload/v123/categorys/test.png"

		category := dbgen.Category{
			ID: id,
			ImageUrl: sql.NullString{
				Valid:  true,
				String: imageURL,
			},
		}

		// 1. GetByID
		deps.repo.EXPECT().
			GetByID(ctx, id).
			Return(category, nil)

		// 2. Delete image di cloudinary
		deps.cloudinary.EXPECT().
			DeleteImage(ctx, gomock.Any()).
			Return(nil)

		// 3. Delete di database
		deps.repo.EXPECT().
			Delete(ctx, id).
			Return(nil)

		err := deps.service.Delete(ctx, id.String())
		assert.NoError(t, err)
	})

	t.Run("fail - invalid uuid", func(t *testing.T) {
		err := deps.service.Delete(ctx, "invalid-uuid")
		assert.Error(t, err)
	})

	t.Run("fail - get by id error", func(t *testing.T) {
		deps.repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.Category{}, errors.New("not found"))

		err := deps.service.Delete(ctx, id.String())
		assert.Error(t, err)
	})
	t.Run("fail - delete image error", func(t *testing.T) {
		imageURL := "https://res.cloudinary.com/demo/image/upload/v123/categorys/test.png"

		category := dbgen.Category{
			ID: id,
			ImageUrl: sql.NullString{
				Valid:  true,
				String: imageURL,
			},
		}

		deps.repo.EXPECT().
			GetByID(ctx, id).
			Return(category, nil)

		deps.cloudinary.EXPECT().
			DeleteImage(ctx, gomock.Any()).
			Return(errors.New("cloudinary error"))

		err := deps.service.Delete(ctx, id.String())
		assert.Error(t, err)
	})

	t.Run("fail - delete db error", func(t *testing.T) {
		category := dbgen.Category{
			ID:       id,
			ImageUrl: sql.NullString{Valid: false},
		}

		deps.repo.EXPECT().
			GetByID(ctx, id).
			Return(category, nil)

		deps.repo.EXPECT().
			Delete(ctx, id).
			Return(errors.New("delete failed"))

		err := deps.service.Delete(ctx, id.String())
		assert.Error(t, err)
	})

}
