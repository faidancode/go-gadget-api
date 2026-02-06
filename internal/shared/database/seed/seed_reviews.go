package seed

import (
	"context"
	"database/sql"
	"fmt"
	"go-gadget-api/internal/shared/database/dbgen"
	"log"
	"math/rand"
)

func SeedReviews(db *sql.DB) error {
	ctx := context.Background()
	q := dbgen.New(db)

	orders, err := q.ListOrdersAdmin(ctx, dbgen.ListOrdersAdminParams{Limit: 20, Offset: 0})
	if err != nil {
		return fmt.Errorf("gagal ambil data orders untuk review: %v", err)
	}

	if len(orders) == 0 {
		log.Println("Warning: Tidak ada order ditemukan. Jalankan SeedOrders dulu.")
		return nil
	}

	comments := []string{
		"Barang sampai dengan selamat, original banget!",
		"Kualitas mantap, sesuai deskripsi dan foto.",
		"Pengiriman agak lama tapi produk sangat memuaskan.",
		"Respon seller cepat, barang dipacking rapi dan aman.",
		"Worth it banget untuk harga segini, fungsi normal.",
		"Build quality-nya premium, gak nyesel beli di sini.",
	}

	log.Println("Memulai seeding review berdasarkan data Order...")

	for _, order := range orders {
		// Ambil item yang ada di dalam order tersebut
		items, err := q.GetOrderItems(ctx, order.ID)
		if err != nil {
			log.Printf("Gagal ambil item untuk order %s: %v\n", order.ID, err)
			continue
		}

		for _, item := range items {
			// Memberikan rating acak 4-5
			rating := int32(rand.Intn(2) + 4)
			comment := comments[rand.Intn(len(comments))]

			_, err := q.CreateReview(ctx, dbgen.CreateReviewParams{
				UserID:             order.UserID,
				ProductID:          item.ProductID,
				OrderID:            order.ID, // Langsung pakai order.ID (tipe uuid.UUID)
				Rating:             rating,
				Comment:            comment, // Langsung pakai string (tipe string)
				IsVerifiedPurchase: true,
			})

			if err != nil {
				log.Printf("Gagal insert review untuk produk %s oleh user %s: %v\n", item.NameSnapshot, order.UserID, err)
				continue
			}

			log.Printf("Berhasil review produk: %s (Rating: %d)\n", item.NameSnapshot, rating)
		}
	}

	log.Println("Seeding reviews selesai!")
	return nil
}
