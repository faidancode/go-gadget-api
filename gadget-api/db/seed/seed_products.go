package seed

import (
	"context"
	"database/sql"
	"fmt"
	"gadget-api/internal/dbgen"
	"log"
	"strings"

	"github.com/google/uuid"
)

// Sekarang mengembalikan error
func SeedAll(db *sql.DB) error {
	ctx := context.Background()
	q := dbgen.New(db)

	categoryMap := make(map[string]uuid.UUID)

	categories := []struct {
		Name string
		Slug string
	}{
		{"Smartphone", "smartphone"},
		{"Laptop and PC", "laptop-and-pc"},
		{"Wearable Gadgets", "wearable-gadgets"},
		{"Accessories", "accessories"},
		{"Tablet", "tablet"},
	}

	for _, cat := range categories {
		c, err := q.CreateCategory(ctx, dbgen.CreateCategoryParams{
			Name:        cat.Name,
			Slug:        cat.Slug,
			Description: sql.NullString{String: "Category for " + cat.Name, Valid: true},
		})

		if err != nil {
			// Jika error karena sudah ada, coba ambil ID yang sudah ada
			existingCat, errGet := q.GetCategoryBySlug(ctx, cat.Slug)
			if errGet != nil {
				return fmt.Errorf("gagal membuat atau mengambil kategori %s: %v", cat.Name, errGet)
			}
			categoryMap[cat.Name] = existingCat.ID
		} else {
			categoryMap[cat.Name] = c.ID
		}
	}

	products := []struct {
		CategoryKey string
		Name        string
		Price       float64
	}{
		{"Smartphone", "iPhone 15 Pro", 18500000},
		{"Smartphone", "Samsung S24 Ultra", 21000000},
		{"Smartphone", "Xiaomi 14", 12000000},
		{"Laptop and PC", "MacBook Pro M3", 25000000},
		{"Laptop and PC", "ASUS ROG Zephyrus", 30000000},
		{"Laptop and PC", "Lenovo Yoga Slim", 15000000},
		{"Wearable Gadgets", "Apple Watch Ultra 2", 15000000},
		{"Wearable Gadgets", "Galaxy Watch 6", 4000000},
		{"Wearable Gadgets", "Garmin Epix Gen 2", 18000000},
		{"Accessories", "Keychron K2 V2", 1200000},
		{"Accessories", "Logitech MX Master 3S", 1500000},
		{"Accessories", "Sony WH-1000XM5", 4500000},
		{"Tablet", "iPad Pro M2", 16000000},
		{"Tablet", "Samsung Tab S9", 13000000},
		{"Tablet", "Huawei MatePad Pro", 9000000},
	}

	for _, p := range products {
		catID, ok := categoryMap[p.CategoryKey]
		if !ok {
			continue
		}

		_, err := q.CreateProduct(ctx, dbgen.CreateProductParams{
			CategoryID:  catID,
			Name:        p.Name,
			Slug:        strings.ToLower(strings.ReplaceAll(p.Name, " ", "-")) + "-" + uuid.New().String()[:4],
			Price:       fmt.Sprintf("%.2f", p.Price),
			Stock:       10,
			Description: sql.NullString{String: "High quality " + p.Name, Valid: true},
			Sku:         sql.NullString{String: "SKU-" + uuid.New().String()[:8], Valid: true},
			ImageUrl:    sql.NullString{String: "https://picsum.photos/400", Valid: true},
		})

		if err != nil {
			log.Printf("Gagal insert produk %s: %v\n", p.Name, err)
			// Kita lanjut saja ke produk berikutnya jika satu gagal
		}
	}

	log.Println("Seeding completed successfully.")
	return nil
}
