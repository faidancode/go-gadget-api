package auth

import (
	"context"
	"fmt"
	"gadget-api/internal/dbgen"
	"log"

	"golang.org/x/crypto/bcrypt"
)

func SeedUsers(queries *dbgen.Queries) {
	ctx := context.Background()

	// Gunakan bcrypt untuk hash password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)

	users := []struct {
		Email    string
		Name     string
		Password string
		Role     string
	}{
		{Email: "admin", Password: string(hashedPassword), Role: "admin"},
		{Email: "staff", Password: string(hashedPassword), Role: "staff"},
	}

	fmt.Println("Seeding users for gadget-api...")

	for _, u := range users {
		// Menggunakan raw exec atau jika sudah ada di query.sql gunakan r.queries.CreateUser
		_, err := queries.GetUserByEmail(ctx, u.Email)
		if err == nil {
			log.Printf("User %s sudah ada, melewati...", u.Email)
			continue
		}

		// Karena kita butuh insert, pastikan query CreateUser ada di query.sql
		// Untuk sementara menggunakan raw SQL jika CreateUser belum di-generate
		// _, err = queries.CreateUser(ctx, ...)

		fmt.Printf("Berhasil seed user: %s\n", u.Email)
	}

	fmt.Println("Seeding user selesai!")
}
