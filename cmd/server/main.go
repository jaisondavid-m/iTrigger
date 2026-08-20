package main

import (
	"log"
	"os"

	"iTrigger/internal/server"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file if present
	_ = godotenv.Load()

	secret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if secret == "" {
		log.Println("[Warning] GITHUB_WEBHOOK_SECRET environment variable is not set. Automatically falling back to project-specific webhook secrets.")
	}

	srv := server.New(secret)

	log.Println("Mini Jenkins starting on :8080")

	if err := srv.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
