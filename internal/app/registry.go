package app

import (
	"database/sql"

	"go-gadget-api/internal/address"
	"go-gadget-api/internal/auth"
	"go-gadget-api/internal/brand"
	"go-gadget-api/internal/cart"
	"go-gadget-api/internal/category"
	"go-gadget-api/internal/cloudinary"
	"go-gadget-api/internal/order"
	"go-gadget-api/internal/outbox"
	"go-gadget-api/internal/product"
	"go-gadget-api/internal/product/adapters"
	"go-gadget-api/internal/review"
	"go-gadget-api/internal/shared/database/dbgen"

	"github.com/gin-gonic/gin"
)

func registerModules(router *gin.Engine, db *sql.DB, cloudinaryService cloudinary.Service) {
	queries := dbgen.New(db)

	// --- Repositories ---
	authRepo := auth.NewRepository(queries)
	categoryRepo := category.NewRepository(queries)
	brandRepo := brand.NewRepository(queries)
	productRepo := product.NewRepository(queries)
	reviewRepo := review.NewRepository(queries)
	cartRepo := cart.NewRepository(queries)
	addressRepo := address.NewRepository(queries)
	orderRepo := order.NewRepository(queries)
	outboxRepo := outbox.NewRepository(queries)

	// --- Services ---
	authService := auth.NewService(authRepo)
	categoryService := category.NewService(db, categoryRepo, cloudinaryService)
	brandService := brand.NewService(db, brandRepo, cloudinaryService)
	reviewService := review.NewService(db, reviewRepo, productRepo)
	productService := product.NewService(db, productRepo, categoryRepo, reviewRepo, cloudinaryService)
	cartService := cart.NewService(db, cartRepo)
	addressService := address.NewService(db, addressRepo)
	orderService := order.NewService(order.Deps{
		DB:         db,
		Repo:       orderRepo,
		OutboxRepo: outboxRepo,
		CartSvc:    cartService,
	})

	// --- Adapters ---
	reviewEligibilityAdapter := adapters.NewReviewEligibilityAdapter(reviewService)

	// --- Handlers ---
	authHandler := auth.NewHandler(authService)
	categoryHandler := category.NewHandler(categoryService)
	brandHandler := brand.NewHandler(brandService)
	reviewHandler := review.NewHandler(reviewService)
	cartHandler := cart.NewHandler(cartService)
	addressHandler := address.NewHandler(addressService)
	productHandler := product.NewHandler(productService, reviewEligibilityAdapter)
	orderHandler := order.NewHandler(orderService)

	// --- Routes Registration ---
	api := router.Group("/api/v1")
	{
		auth.RegisterRoutes(api, authHandler)
		brand.RegisterRoutes(api, brandHandler)
		category.RegisterRoutes(api, categoryHandler)
		product.RegisterRoutes(api, productHandler)
		review.RegisterRoutes(api, reviewHandler)
		cart.RegisterRoutes(api, cartHandler)
		address.RegisterRoutes(api, addressHandler)
		order.RegisterRoutes(api, orderHandler)
	}
}
