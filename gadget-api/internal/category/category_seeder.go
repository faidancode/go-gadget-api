package category

import (
	"context"
	"fmt"
	"gadget-api/internal/dbgen" // Menggunakan package gadget-api
	"log"
	"strings"
)

func SeedCategories(repo Repository) {
	ctx := context.Background()

	categories := []CreateCategoryRequest{
		{Name: "Handphone", Description: "Smartphone dan ponsel fitur terbaru", ImageUrl: "https://example.com/hp.jpg"},
		{Name: "Tablet", Description: "Tablet Android dan iPad untuk produktivitas", ImageUrl: "https://example.com/tablet.jpg"},
		{Name: "Laptop", Description: "Laptop gaming, kantor, dan ultrabook", ImageUrl: "https://example.com/laptop.jpg"},
		{Name: "Wearable", Description: "Smartwatch dan TWS", ImageUrl: "https://example.com/wearable.jpg"},
		{Name: "Accessories", Description: "Charger, kabel, dan casing", ImageUrl: "https://example.com/acc.jpg"},
	}

	fmt.Println("Seeding categories for gadget-api...")

	for _, cat := range categories {
		// Logika slug sederhana: Handphone -> handphone
		slug := strings.ToLower(cat.Name)

		_, err := repo.Create(ctx, dbgen.CreateCategoryParams{
			Name:        cat.Name,
			Slug:        slug,
			Description: dbgen.NewNullString(cat.Description),
			ImageUrl:    dbgen.NewNullString(cat.ImageUrl),
		})

		if err != nil {
			log.Printf("Gagal seed kategori %s: %v", cat.Name, err)
			continue
		}
		fmt.Printf("Berhasil seed: %s\n", cat.Name)
	}

	fmt.Println("Seeding selesai!")
}
