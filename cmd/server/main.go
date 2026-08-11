package main

import (
	"log"
	"os"

	"iTrigger/internal/server"
)

func main() {
	secret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if secret == "" {
		log.Fatal("GITHUB_WEBHOOK_SECRET must be set")
	}

	srv := server.New(secret)

	log.Println("Mini Jenkins starting on :8080")

	if err := srv.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
