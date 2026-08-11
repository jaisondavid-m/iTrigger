package server

import (
	"encoding/json"
	"net/http"

	"iTrigger/internal/models"
	"iTrigger/internal/webhook"
)

type Server struct {
	mux *http.ServeMux
}

func New(secret string) *Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/healthz", healthHandler)
	mux.Handle("/api/webhooks/github", webhook.New(secret))

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
