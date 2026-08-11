package webhook

import (
	"encoding/json"
	"log"
	"net/http"
)

type Payload struct {
	Repository string `json:"repository"`
	Branch     string `json:"branch"`
}

func Handler(w http.ResponseWriter, r *http.Request) {

	// Only allow POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload Payload

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf(
		"Webhook received: repository=%s branch=%s",
		payload.Repository,
		payload.Branch,
	)

	w.Header().Set("Content-Type", "application/json")

	response := map[string]string{
		"status":  "received",
		"message": "Deployment trigger received",
	}

	json.NewEncoder(w).Encode(response)
}