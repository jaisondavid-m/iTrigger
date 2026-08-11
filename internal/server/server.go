package server

import (
	"net/http"

	"iTrigger/internal/webhook"
)

type Server struct {
	mux *http.ServeMux
}

func New() *Server {
	mux := http.NewServeMux()

	// Webhook endpoint
	mux.HandleFunc("/webhook", webhook.Handler)

	return &Server{
		mux: mux,
	}
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}