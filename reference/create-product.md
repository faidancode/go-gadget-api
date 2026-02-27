## Queries
type CreateProductParams struct {
	CategoryID  uuid.UUID      `json:"category_id"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Description sql.NullString `json:"description"`
	Price       string         `json:"price"`
	Stock       int32          `json:"stock"`
	Sku         sql.NullString `json:"sku"`
	ImageUrl    sql.NullString `json:"image_url"`
}

## Repo
func (r *repository) Create(ctx context.Context, arg dbgen.CreateProductParams) (dbgen.Product, error) {
	return r.queries.CreateProduct(ctx, arg)
}

## Service

type service struct {
	db           *sql.DB
	repo         Repository
	categoryRepo category.Repository
	reviewRepo   ReviewRepository
}

func NewService(db *sql.DB, repo Repository, categoryRepo category.Repository, reviewRepo ReviewRepository) Service {
	return &service{
		db:           db,
		repo:         repo,
		categoryRepo: categoryRepo,
		reviewRepo:   reviewRepo,
	}
}


func (s *service) Create(ctx context.Context, req CreateProductRequest) (ProductAdminResponse, error) {
	catID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return ProductAdminResponse{}, fmt.Errorf("invalid category id")
	}

	_, err = s.categoryRepo.GetByID(ctx, catID)
	if err != nil {
		return ProductAdminResponse{}, fmt.Errorf("category not found")
	}

	slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-")) + "-" + uuid.New().String()[:5]
	priceStr := fmt.Sprintf("%.2f", req.Price)

	p, err := s.repo.Create(ctx, dbgen.CreateProductParams{
		CategoryID:  catID,
		Name:        req.Name,
		Slug:        slug,
		Description: dbgen.NewNullString(req.Description),
		Price:       priceStr,
		Stock:       req.Stock,
		Sku:         dbgen.NewNullString(req.SKU),
		ImageUrl:    dbgen.NewNullString(req.ImageUrl),
	})

	if err != nil {
		return ProductAdminResponse{}, err
	}

	return s.GetByIDAdmin(ctx, p.ID.String())
}

func (s *service) Update(ctx context.Context, idStr string, req UpdateProductRequest) (ProductAdminResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return ProductAdminResponse{}, fmt.Errorf("invalid product id")
	}

	// 1. Cek apakah produk ada
	existingProduct, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ProductAdminResponse{}, fmt.Errorf("product not found")
	}

	// 2. Siapkan params untuk repo
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

	// 3. Update hanya field yang dikirim (Patch-like behavior)
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

	// 4. Eksekusi Update ke Repo
	_, err = s.repo.Update(ctx, params)
	if err != nil {
		return ProductAdminResponse{}, err
	}

	// 5. Kembalikan data terbaru
	return s.GetByIDAdmin(ctx, idStr)
}

## Controller
func (ctrl *Controller) Create(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Input tidak valid",
			err.Error(),
		)
		return
	}

	res, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(
			c,
			http.StatusInternalServerError,
			"CREATE_ERROR",
			"Gagal membuat produk",
			err.Error(),
		)
		return
	}

	response.Success(c, http.StatusCreated, res, nil)
}

func (ctrl *Controller) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Input tidak valid",
			err.Error(),
		)
		return
	}

	res, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "product not found" || err.Error() == "category not found" {
			statusCode = http.StatusNotFound
		}

		response.Error(
			c,
			statusCode,
			"UPDATE_ERROR",
			"Gagal memperbarui produk",
			err.Error(),
		)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}


## Service Test

func TestService_Create(t *testing.T) {
	ctrl := gomock.NewHandler(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	catRepo := catMock.NewMockRepository(ctrl)
	service := NewService(repo, catRepo)

	ctx := context.Background()
	catID := uuid.New()

	req := CreateProductRequest{
		CategoryID: catID.String(),
		Name:       "iPhone 15",
		Price:      15000000,
		Stock:      10,
	}

	t.Run("Success", func(t *testing.T) {
		repoID := uuid.New()

		catRepo.EXPECT().
			GetByID(ctx, catID).
			Return(dbgen.Category{ID: catID}, nil)

		repo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(dbgen.Product{ID: repoID}, nil)

		repo.EXPECT().
			GetByID(ctx, repoID).
			Return(dbgen.GetProductByIDRow{
				ID:           repoID,
				Name:         req.Name,
				Price:        "15000000.00",
				Stock:        10,
				IsActive:     dbgen.NewNullBool(true),
				CategoryName: "Phone",
				CreatedAt:    time.Now(),
			}, nil)

		res, err := service.Create(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, req.Name, res.Name)
	})

	t.Run("Invalid Category ID", func(t *testing.T) {
		_, err := service.Create(ctx, CreateProductRequest{
			CategoryID: "invalid-uuid",
		})

		assert.Error(t, err)
		assert.Equal(t, "invalid category id", err.Error())
	})

	t.Run("Category Not Found", func(t *testing.T) {
		catRepo.EXPECT().
			GetByID(ctx, catID).
			Return(dbgen.Category{}, errors.New("not found"))

		_, err := service.Create(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "category not found", err.Error())
	})
}

func TestService_Update(t *testing.T) {
	ctrl := gomock.NewHandler(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	service := NewService(repo, nil)

	ctx := context.Background()
	id := uuid.New()

	existing := dbgen.GetProductByIDRow{
		ID:    id,
		Name:  "Old Name",
		Price: "100.00",
		Stock: 5,
	}

	req := UpdateProductRequest{
		Name:  "New Name",
		Price: 200,
	}

	t.Run("Success", func(t *testing.T) {
		repo.EXPECT().
			GetByID(ctx, id).
			Return(existing, nil)

		repo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(dbgen.Product{}, nil)

		repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.GetProductByIDRow{
				ID:    id,
				Name:  req.Name,
				Price: "200.00",
				Stock: 5,
			}, nil)

		res, err := service.Update(ctx, id.String(), req)

		assert.NoError(t, err)
		assert.Equal(t, req.Name, res.Name)
	})

	t.Run("Product Not Found", func(t *testing.T) {
		repo.EXPECT().
			GetByID(ctx, id).
			Return(dbgen.GetProductByIDRow{}, errors.New("not found"))

		_, err := service.Update(ctx, id.String(), req)

		assert.Error(t, err)
		assert.Equal(t, "product not found", err.Error())
	})
}

## Controller Test

func TestCreateProduct(t *testing.T) {
	router, svc := setupTest()

	payload := CreateProductRequest{
		Name:       "Macbook",
		Price:      20000000,
		Stock:      10,
		CategoryID: uuid.New().String(),
	}

	t.Run("Success", func(t *testing.T) {
		svc.CreateFn = func(ctx context.Context, req CreateProductRequest) (ProductAdminResponse, error) {
			return ProductAdminResponse{Name: req.Name}, nil
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Validation Error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		svc.CreateFn = func(ctx context.Context, req CreateProductRequest) (ProductAdminResponse, error) {
			return ProductAdminResponse{}, errors.New("create failed")
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUpdateProduct(t *testing.T) {
	router, svc := setupTest()
	id := uuid.New().String()

	payload := UpdateProductRequest{
		Name:  "Updated",
		Price: 9999,
	}

	t.Run("Success", func(t *testing.T) {
		svc.UpdateFn = func(ctx context.Context, pid string, req UpdateProductRequest) (ProductAdminResponse, error) {
			return ProductAdminResponse{Name: req.Name}, nil
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPut, "/products/"+id, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		svc.UpdateFn = func(ctx context.Context, pid string, req UpdateProductRequest) (ProductAdminResponse, error) {
			return ProductAdminResponse{}, errors.New("product not found")
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPut, "/products/"+id, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}


## Task
1. Buatkan functio upload image ke cloudinary
2. Tambahkan ini pada service test, ini sebagai contoh saja:
ctrl := gomock.NewHandler(t)
	defer ctrl.Finish()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)

	// Sekarang menyertakan DB untuk keperluan transaksi
	svc := order.NewService(db, orderRepo, cartSvc)
	ctx := context.Background()

3. Tambahkan ini pada controller test:

// ==================== HELPER FUNCTIONS ====================

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func newTestController(svc order.Service) *order.Handler {
	return order.NewHandler(svc)
}