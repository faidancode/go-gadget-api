package main

import (
	"database/sql"
	"log"
	"os"

	"gadget-api/internal/auth"
	"gadget-api/internal/category"
	"gadget-api/internal/cloudinary"
	"gadget-api/internal/dbgen"
	"gadget-api/internal/product"
	"gadget-api/internal/review"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// 1. Load Environment
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// 2. Database Connection
	db, err := sql.Open("postgres", os.Getenv("DB"))
	if err != nil {
		log.Fatal("Cannot connect to database:", err)
	}
	defer db.Close()

	// 3. Initialize SQLC Queries
	queries := dbgen.New(db)

	// ===== Setup Cloudinary =====
	cloudinaryService, err := cloudinary.NewService(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
		"gadget-store/products", // folder name
	)
	if err != nil {
		log.Fatal("Failed to initialize Cloudinary:", err)
	}

	// 4. Initialize Modules (Dependency Injection)

	authRepo := auth.NewRepository(queries)
	authService := auth.NewService(authRepo)
	authController := auth.NewController(authService)

	categoryRepo := category.NewRepository(queries)
	categoryService := category.NewService(categoryRepo)
	categoryController := category.NewController(categoryService)

	productRepo := product.NewRepository(queries)

	reviewRepo := review.NewRepository(queries)
	reviewService := review.NewService(db, reviewRepo, productRepo)
	reviewController := review.NewController(reviewService)

	productService := product.NewService(db, productRepo, categoryRepo, reviewRepo, cloudinaryService)
	productController := product.NewController(productService)

	registry := ControllerRegistry{
		Auth:     authController,
		Category: categoryController,
		Product:  productController,
		Review:   reviewController,
	}

	// 4. Jalankan Router
	r := gin.Default()
	setupRoutes(r, registry)

	// 7. Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	r.Run(":" + port)
}
