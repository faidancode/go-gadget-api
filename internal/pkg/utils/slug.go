// internal/pkg/utils/slug.go atau direktori helper Anda
package utils

import (
	"log"
	"regexp"
	"strings"
)

func GenerateSlug(s string) string {
	log.Println("GEN SLUG")
	// 1. Ubah ke lowercase
	slug := strings.ToLower(s)

	// 2. Ganti karakter non-alfanumerik (kecuali spasi) dengan spasi
	reg, _ := regexp.Compile("[^a-z0-9]+")
	slug = reg.ReplaceAllString(slug, " ")

	// 3. Trim spasi di awal dan akhir
	slug = strings.TrimSpace(slug)

	// 4. Ganti spasi dengan dash (-)
	slug = strings.ReplaceAll(slug, " ", "-")

	// 5. Tangani double dash jika ada
	regDoubleDash, _ := regexp.Compile("-+")
	slug = regDoubleDash.ReplaceAllString(slug, "-")

	return slug
}
