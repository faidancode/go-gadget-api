package main

import (
	"database/sql"
	"log"
	"os"

	"gadget-api/internal/auth"
	"gadget-api/internal/category"
	"gadget-api/internal/dbgen"
	"gadget-api/internal/product"

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

	// 4. Initialize Modules (Dependency Injection)

	authRepo := auth.NewRepository(queries)
	authService := auth.NewService(authRepo)
	authController := auth.NewController(authService)

	catRepo := category.NewRepository(queries)
	catService := category.NewService(catRepo)
	catController := category.NewController(catService)

	prodRepo := product.NewRepository(queries)
	prodService := product.NewService(prodRepo, catRepo)
	prodController := product.NewController(prodService)

	registry := ControllerRegistry{
		Auth:     authController,
		Category: catController,
		Product:  prodController,
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
