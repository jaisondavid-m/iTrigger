package server

import (
	"embed"
	"net/http"

	"iTrigger/internal/routes"
)

//go:embed web/*
var webFS embed.FS

type Server struct {
	mux *http.ServeMux
}

func New(secret string) *Server {
	mux := http.NewServeMux()
	routes.Register(mux, secret, webFS)

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

