package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "data/itrigger.db")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT username, is_admin, can_create_project FROM users")
	if err != nil {
		log.Fatalf("failed to query users table: %v", err)
	}
	defer rows.Close()

	fmt.Println("Users in database:")
	for rows.Next() {
		var username string
		var isAdmin, canCreate int
		if err := rows.Scan(&username, &isAdmin, &canCreate); err == nil {
			fmt.Printf("- Username: %q, IsAdmin: %v, CanCreate: %v\n", username, isAdmin, canCreate)
		} else {
			fmt.Printf("Scan error: %v\n", err)
		}
	}
}
