package seed

import (
	"context"
	"database/sql"
	"log"

	"go-gadget-api/internal/pkg/security"
	"go-gadget-api/internal/shared/database/dbgen"
)

func SeedUsers(db *sql.DB) error {
	ctx := context.Background()

	// ✅ dbgen adalah hasil sqlc
	q := dbgen.New(db)

	users := []struct {
		Email    string
		Name     string
		Password string
		Role     string
	}{
		{
			Email:    "adminone@example.com",
			Name:     "Admin One",
			Password: "admin23#",
			Role:     "ADMIN",
		},
		{
			Email:    "supadmin@example.com",
			Name:     "Super Admin",
			Password: "ssadmin23#",
			Role:     "SUPERADMIN",
		},
	}

	for _, u := range users {
		hashed, err := security.HashPassword(u.Password)
		if err != nil {
			return err
		}

		_, err = q.CreateUser(ctx, dbgen.CreateUserParams{
			Email:    u.Email,
			Name:     u.Name,
			Password: hashed,
			Role:     u.Role,
		})

		if err != nil {
			// biasanya duplicate email
			log.Println("skip seed user:", err)
			continue
		}
	}

	return nil
}
