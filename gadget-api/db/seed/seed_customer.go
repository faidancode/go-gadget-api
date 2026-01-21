package seed

import (
	"context"
	"database/sql"
	"log"

	"gadget-api/internal/dbgen"
	"gadget-api/internal/pkg/security"
)

func SeedCustomers(db *sql.DB) error {
	ctx := context.Background()
	q := dbgen.New(db)

	customers := []struct {
		Email    string
		Name     string
		Password string
	}{
		{
			Email:    "customergg1@yopmail.com",
			Name:     "Customer One",
			Password: "customer123",
		},
		{
			Email:    "customergg2@yopmail.com",
			Name:     "Customer Two",
			Password: "customer123",
		},
		{
			Email:    "customergg3@yopmail.com",
			Name:     "Customer Three",
			Password: "customer123",
		},
		{
			Email:    "customergg4@yopmail.com",
			Name:     "Customer Four",
			Password: "customer123",
		},
		{
			Email:    "customergg5@yopmail.com",
			Name:     "Customer Five",
			Password: "customer123",
		},
	}

	for _, c := range customers {
		hashed, err := security.HashPassword(c.Password)
		if err != nil {
			return err
		}

		_, err = q.CreateUser(ctx, dbgen.CreateUserParams{
			Email:    c.Email,
			Name:     c.Name,
			Password: hashed,
			Role:     "CUSTOMER",
		})

		if err != nil {
			log.Println("skip seed customer:", err)
			continue
		}
	}

	return nil
}
