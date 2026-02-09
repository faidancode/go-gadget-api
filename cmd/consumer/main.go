package main

import (
	"go-gadget-api/internal/app"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	if err := app.RunConsumer(); err != nil {
		log.Fatal(err)
	}
}
