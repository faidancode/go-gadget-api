package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-gadget-api/internal/dbgen"
	"log"
	"math/rand"
	"strconv"
	"time"
)

func SeedOrders(db *sql.DB) error {
	ctx := context.Background()
	q := dbgen.New(db)

	customers := []string{
		"customergg1@yopmail.com",
		"customergg2@yopmail.com",
		"customergg3@yopmail.com",
		"customergg4@yopmail.com",
		"customergg5@yopmail.com",
	}

	products, err := q.ListProductsForInternal(ctx)
	if err != nil || len(products) == 0 {
		return fmt.Errorf("seeder order butuh data produk")
	}

	log.Println("Memulai seeding orders (min 4 customer per product)...")

	for _, p := range products {
		priceFloat, err := strconv.ParseFloat(p.Price, 64)
		if err != nil {
			continue
		}

		// shuffle customer
		rand.Shuffle(len(customers), func(i, j int) {
			customers[i], customers[j] = customers[j], customers[i]
		})

		selected := customers[:4]

		for _, email := range selected {
			user, err := q.GetUserByEmail(ctx, email)
			if err != nil {
				continue
			}

			qty := int32(1)
			subtotal := priceFloat
			shipping := 15000.0
			total := subtotal + shipping

			order, err := q.CreateOrder(ctx, dbgen.CreateOrderParams{
				OrderNumber: fmt.Sprintf("ORD-%d", time.Now().UnixNano()),
				UserID:      user.ID,
				Status:      "COMPLETED",

				SubtotalPrice: fmt.Sprintf("%.2f", subtotal),
				ShippingPrice: fmt.Sprintf("%.2f", shipping),
				TotalPrice:    fmt.Sprintf("%.2f", total),

				AddressSnapshot: json.RawMessage(`{"address_id":"seed"}`),
			})
			if err != nil {
				continue
			}

			_ = q.CreateOrderItem(ctx, dbgen.CreateOrderItemParams{
				OrderID:      order.ID,
				ProductID:    p.ID,
				NameSnapshot: p.Name,
				UnitPrice:    fmt.Sprintf("%.2f", priceFloat),
				Quantity:     qty,
				TotalPrice:   fmt.Sprintf("%.2f", subtotal),
			})
		}
	}

	log.Println("Seeding orders selesai")
	return nil
}
