package app

import (
	"database/sql"

	"go-gadget-api/internal/address"
	"go-gadget-api/internal/auth"
	"go-gadget-api/internal/brand"
	"go-gadget-api/internal/cart"
	"go-gadget-api/internal/category"
	"go-gadget-api/internal/cloudinary"
	"go-gadget-api/internal/customer"
	"go-gadget-api/internal/email"
	"go-gadget-api/internal/order"
	"go-gadget-api/internal/outbox"
	"go-gadget-api/internal/product"
	"go-gadget-api/internal/product/adapters"
	"go-gadget-api/internal/review"
	"go-gadget-api/internal/shared/database/dbgen"
	"go-gadget-api/internal/wishlist"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func registerModules(
	router *gin.Engine,
	db *sql.DB,
	rdb *redis.Client,
	cloudinaryService cloudinary.Service,
	logger *zap.Logger,
) {
	queries := dbgen.New(db)

	// --- Repositories ---
	authRepo := auth.NewRepository(queries, db)
	categoryRepo := category.NewRepository(queries)
	brandRepo := brand.NewRepository(queries)
	productRepo := product.NewRepository(queries)
	reviewRepo := review.NewRepository(queries)
	cartRepo := cart.NewRepository(queries)
	addressRepo := address.NewRepository(queries)
	orderRepo := order.NewRepository(queries)
	outboxRepo := outbox.NewRepository(queries)
	customerRepo := customer.NewRepository(queries)
	wishlistRepo := wishlist.NewRepository(queries)

	// --- Services ---
	emailService, err := email.NewResendServiceFromEnv()
	if err != nil {
		panic(err)
	}

	authService := auth.NewService(authRepo, emailService)
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
	customerService := customer.NewService(db, customerRepo)
	wishlistService := wishlist.NewService(db, wishlistRepo)

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
	orderHandler := order.NewHandler(orderService, rdb)
	customerHandler := customer.NewHandler(customerService)
	wishlistHandler := wishlist.NewHandler(wishlistService)

	// --- Routes Registration ---
	api := router.Group("/api/v1")
	{
		auth.RegisterRoutes(api, authHandler)
		brand.RegisterRoutes(api, brandHandler)
		category.RegisterRoutes(api, categoryHandler)
		product.RegisterRoutes(api, productHandler)
		review.RegisterRoutes(api, reviewHandler)
		cart.RegisterRoutes(api, cartHandler, logger)
		address.RegisterRoutes(api, addressHandler)
		order.RegisterRoutes(api, orderHandler, rdb, logger)
		customer.RegisterRoutes(api, customerHandler)
		wishlist.RegisterRoutes(api, wishlistHandler, logger)
	}
}
