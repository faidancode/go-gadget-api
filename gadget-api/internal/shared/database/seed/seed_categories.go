package seed

import (
	"context"
	"database/sql"
	"log"
	"strings"

	"gadget-api/internal/dbgen"
)

func SeedCategories(db *sql.DB) error {
	ctx := context.Background()
	q := dbgen.New(db)

	categories := []struct {
		Name        string
		Description string
		ImageURL    string
	}{
		{
			Name:        "Smartphone",
			Description: "Perangkat telepon pintar terbaru dari berbagai brand ternama.",
			ImageURL:    "https://example.com/images/categories/smartphones.jpg",
		},
		{
			Name:        "Laptop & PC",
			Description: "Laptop gaming, ultrabook, dan perangkat komputer untuk produktivitas.",
			ImageURL:    "https://example.com/images/categories/laptops.jpg",
		},
		{
			Name:        "Wearable Gadgets",
			Description: "Smartwatch, TWS, dan perangkat wearable lainnya.",
			ImageURL:    "https://example.com/images/categories/wearables.jpg",
		},
		{
			Name:        "Accessories",
			Description: "Charger, kabel data, casing, dan aksesoris gadget lainnya.",
			ImageURL:    "https://example.com/images/categories/accessories.jpg",
		},
		{
			Name:        "Tablet",
			Description: "Tablet untuk kebutuhan desain, belajar, dan hiburan.",
			ImageURL:    "https://example.com/images/categories/tablets.jpg",
		},
	}

	for _, cat := range categories {
		// Sederhana: Membuat slug dari nama (Contoh: "Laptop & PC" -> "laptop--pc")
		slug := strings.ToLower(strings.ReplaceAll(cat.Name, " ", "-"))

		_, err := q.CreateCategory(ctx, dbgen.CreateCategoryParams{
			Name:        cat.Name,
			Slug:        slug,
			Description: sql.NullString{String: cat.Description, Valid: true},
			ImageUrl:    sql.NullString{String: cat.ImageURL, Valid: true},
		})

		if err != nil {
			log.Printf("Gagal atau skip seed category %s: %v\n", cat.Name, err)
			continue
		}
	}

	log.Println("Seeding categories completed.")
	return nil
}
