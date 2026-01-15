package main

import (
	"database/sql"
	"log"
	"os"

	"gadget-api/internal/category"
	"gadget-api/internal/dbgen"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Database connection
	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize sqlc queries
	queries := dbgen.New(db)

	// Initialize Repositories
	catRepo := category.NewRepository(queries)

	schema, err := os.ReadFile("db/schema.sql")
	if err != nil {
		log.Fatal("Gagal membaca file schema.sql: ", err)
	}

	_, err = db.Exec(string(schema))
	if err != nil {
		log.Printf("Peringatan: Gagal eksekusi schema (mungkin tabel sudah ada): %v", err)
	}

	// Run Seeders
	category.SeedCategories(catRepo)
}
