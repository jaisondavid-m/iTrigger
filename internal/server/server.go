package server

import (
	"net/http"

	"iTrigger/internal/webhook"
)

type Server struct {
	mux *http.ServeMux
}

func New(secret string) *Server {
	mux := http.NewServeMux()

	mux.Handle("/api/webhooks/github", webhook.New(secret))

	return &Server{
		mux: mux,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}
