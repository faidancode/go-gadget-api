package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	"gadget-api/internal/auth"
	"gadget-api/internal/bootstrap"
	"gadget-api/internal/brand"
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
	// Load env
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// DB
	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal("Cannot connect to database:", err)
	}
	defer db.Close()

	queries := dbgen.New(db)

	cloudinaryService, err := cloudinary.NewService(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
		"gadget-store/products",
	)
	if err != nil {
		log.Fatal("Failed to initialize Cloudinary:", err)
	}

	// DI
	authController := auth.NewController(
		auth.NewService(auth.NewRepository(queries)),
	)

	categoryRepo := category.NewRepository(queries)
	categoryController := category.NewController(
		category.NewService(db, categoryRepo, cloudinaryService),
	)

	brandRepo := brand.NewRepository(queries)
	brandController := brand.NewController(
		brand.NewService(db, brandRepo, cloudinaryService),
	)

	productRepo := product.NewRepository(queries)

	reviewController := review.NewController(
		review.NewService(db, review.NewRepository(queries), productRepo),
	)

	productController := product.NewController(
		product.NewService(db, productRepo, categoryRepo, review.NewRepository(queries), cloudinaryService),
	)

	registry := ControllerRegistry{
		Auth:     authController,
		Brand:    brandController,
		Category: categoryController,
		Product:  productController,
		Review:   reviewController,
	}

	// Router
	r := gin.Default()
	setupRoutes(r, registry)

	// Audit logger
	auditLogger := bootstrap.NewStdoutAuditLogger()

	// Server config
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	bootstrap.StartHTTPServer(
		r,
		bootstrap.ServerConfig{
			Port:         port,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		auditLogger,
	)
}
