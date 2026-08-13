package routes

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

	"iTrigger/internal/models"
	"iTrigger/internal/webhook"
)

// Register declares and maps all application HTTP routes onto the provided ServeMux.
func Register(mux *http.ServeMux, secret string, webFS fs.FS) {
	store := webhook.NewStore()
	webhookHandler := webhook.New(secret, store)

	// Health check endpoints
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/healthz", healthHandler)

	// Webhook API endpoints
	mux.Handle("/api/webhooks/github", webhookHandler)
	mux.HandleFunc("/api/webhooks", webhookHandler.GetEventsHandler)
	mux.HandleFunc("/api/webhooks/clear", webhookHandler.ClearEventsHandler)

	// Static web interface
	if webFS != nil {
		subFS, err := fs.Sub(webFS, "web")
		if err != nil {
			log.Fatalf("failed to sub embed filesystem: %v", err)
		}
		mux.Handle("/", http.FileServer(http.FS(subFS)))
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(models.HealthResponse{Status: "ok"})
}
