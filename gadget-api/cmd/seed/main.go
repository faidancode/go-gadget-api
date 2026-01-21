package main

import (
	"database/sql"
	"log"
	"os"

	"gadget-api/db/seed"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}
	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal("Cannot connect to database:", err)
	}
	defer db.Close()

	// if err := seed.SeedUsers(db); err != nil {
	// 	log.Fatal(err)
	// }

	// if err := seed.SeedCustomers(db); err != nil {
	// 	log.Fatal(err)
	// }

	if err := seed.SeedCategories(db); err != nil {
		log.Fatal(err)
	}

}
