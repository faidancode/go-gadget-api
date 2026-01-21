package brand_test

import (
	"context"
	"database/sql"
	"mime/multipart"
	"testing"

	"gadget-api/internal/brand"
	"gadget-api/internal/dbgen"

	brandMock "gadget-api/internal/mock/brand"
	cloudinaryMock "gadget-api/internal/mock/cloudinary"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ======================= HELPERS =======================

type serviceDeps struct {
	db         *sql.DB
	sqlMock    sqlmock.Sqlmock
	service    brand.Service
	repo       *brandMock.MockRepository
	cloudinary *cloudinaryMock.MockService
}

func setupServiceTest(t *testing.T) *serviceDeps {
	t.Helper()

	ctrl := gomock.NewController(t)
	db, sqlMock, _ := sqlmock.New()

	repo := brandMock.NewMockRepository(ctrl)
	cloudinary := cloudinaryMock.NewMockService(ctrl)

	// Sesuaikan dengan constructor Brand Service Anda yang baru
	svc := brand.NewService(db, repo, cloudinary)

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

// ======================= CREATE =======================

func TestBrandService_Create(t *testing.T) {
	deps := setupServiceTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	brandID := uuid.New()
	req := brand.CreateBrandRequest{
		Name:        "Apple",
		Description: "Premium Tech Brand",
	}

	t.Run("positive - success with image upload", func(t *testing.T) {
		fakeFile := &mockFile{}
		filename := "logo.png"
		imgURL := "https://cloudinary.com/apple.png"

		expectTx(t, deps.sqlMock, true)

		deps.repo.EXPECT().WithTx(gomock.Any()).Return(deps.repo)

		// 1. Mock Create awal
		deps.repo.EXPECT().Create(ctx, gomock.Any()).Return(dbgen.Brand{
			ID:   brandID,
			Name: req.Name,
		}, nil)

		// 2. Mock Cloudinary Upload
		deps.cloudinary.EXPECT().
			UploadImage(ctx, fakeFile, gomock.Any()).
			Return(imgURL, nil)

		// 3. Mock Update dengan ImageUrl
		deps.repo.EXPECT().Update(ctx, gomock.Any()).Return(dbgen.Brand{
			ID:   brandID,
			Name: req.Name,
		}, nil)

		// --- TAMBAHKAN BAGIAN INI ---
		// Karena service memanggil s.GetByID() sebelum return
		deps.repo.EXPECT().
			GetByID(ctx, brandID).
			Return(dbgen.Brand{
				ID:          brandID,
				Name:        req.Name,
				Description: dbgen.NewNullString(req.Description),
				ImageUrl:    dbgen.NewNullString(imgURL),
			}, nil)
		// ----------------------------

		res, err := deps.service.Create(ctx, req, fakeFile, filename)

		assert.NoError(t, err)
		assert.Equal(t, req.Name, res.Name)
		assert.Equal(t, imgURL, res.ImageUrl) // Sekarang ini tidak akan "" lagi
	})

	t.Run("invalid uuid", func(t *testing.T) {
		_, err := deps.service.GetByID(ctx, "invalid-id")
		assert.Error(t, err)
	})
}

// ======================= UPDATE =======================

func TestBrandService_Update(t *testing.T) {
	deps := setupServiceTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	id := uuid.New()
	req := brand.CreateBrandRequest{Name: "Updated Apple"}

	t.Run("success without image change", func(t *testing.T) {
		deps.repo.EXPECT().Update(ctx, gomock.Any()).Return(dbgen.Brand{
			ID:   id,
			Name: req.Name,
		}, nil)

		res, err := deps.service.Update(ctx, id.String(), req)

		assert.NoError(t, err)
		assert.Equal(t, req.Name, res.Name)
	})
}

// ======================= DELETE =======================

func TestBrandService_Delete(t *testing.T) {
	deps := setupServiceTest(t)
	defer deps.db.Close()

	ctx := context.Background()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		deps.repo.EXPECT().Delete(ctx, id).Return(nil)

		err := deps.service.Delete(ctx, id.String())
		assert.NoError(t, err)
	})
}
