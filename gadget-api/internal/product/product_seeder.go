package product

import (
	"context"
	"fmt"
	"gadget-api/internal/dbgen"
	"log"

	"github.com/google/uuid"
)

func SeedProducts(repo Repository, categoryID string, name string, price float64) {
	ctx := context.Background()
	catUUID, _ := uuid.Parse(categoryID)

	_, err := repo.Create(ctx, dbgen.CreateProductParams{
		CategoryID: catUUID,
		Name:       name,
		Slug:       fmt.Sprintf("%s-%s", name, uuid.New().String()[:4]),
		Price:      fmt.Sprintf("%.2f", price), // Konversi float ke string untuk DECIMAL
		Stock:      10,
		Sku:        dbgen.NewNullString("SKU-" + name),
	})

	if err != nil {
		log.Printf("Gagal seed produk %s: %v", name, err)
	} else {
		fmt.Printf("Berhasil seed produk: %s\n", name)
	}
}
