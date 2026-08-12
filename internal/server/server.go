package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

	"iTrigger/internal/models"
	"iTrigger/internal/webhook"
)

//go:embed web/*
var webFS embed.FS

type Server struct {
	mux *http.ServeMux
}

func New(secret string) *Server {
	mux := http.NewServeMux()
	store := webhook.NewStore()
	webhookHandler := webhook.New(secret, store)

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/healthz", healthHandler)
	mux.Handle("/api/webhooks/github", webhookHandler)
	mux.HandleFunc("/api/webhooks", webhookHandler.GetEventsHandler)
	mux.HandleFunc("/api/webhooks/clear", webhookHandler.ClearEventsHandler)


	subFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("failed to sub embed filesystem: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(subFS)))

	return &Server{
		mux: mux,
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

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}

