package main

import (
	"log"

	"iTrigger/internal/server"
)

func main() {
	srv := server.New()

	log.Println("Mini Jenkins starting on :8080")

	if err := srv.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}