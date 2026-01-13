package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"github.com/bak1an/artf/server/handler"
	"go.etcd.io/bbolt"
)

type Server struct {
	listener net.Listener
	dataDir  string
	db       *bbolt.DB
	srv      *http.Server
}

func NewServer(dataDir string, listener net.Listener, db *bbolt.DB) (*Server, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /ping", handler.Ping)
	mux.HandleFunc("GET /version", handler.Version)

	srv := &http.Server{
		Handler: mux,
	}

	return &Server{listener: listener, db: db, srv: srv}, nil
}

func (s *Server) Serve() error {
	slog.Info("serving", "address", s.listener.Addr())
	return s.srv.Serve(s.listener)
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down", "address", s.listener.Addr())
	return s.srv.Shutdown(ctx)
}
