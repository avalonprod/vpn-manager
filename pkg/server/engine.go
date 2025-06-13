package server

import (
	"context"
	"log"
	"net/http"
	"strings"
)

type Server struct {
	httpServer *http.Server
}

type HttpConfig struct {
	Port string
}

func NewServer(cfg *HttpConfig, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:    ":" + cfg.Port,
			Handler: handler,
		},
	}
}

func (s *Server) Run() error {
	port := strings.Replace(s.httpServer.Addr, ":", "", 1)

	log.Printf("Server has ben started on port: %s", port)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
