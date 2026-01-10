package admin

import (
	"log/slog"
	"net"
	"net/http"

	"github.com/bak1an/artf/handler"
	"go.etcd.io/bbolt"
)

type AdminServer struct {
	listener net.Listener
	db       *bbolt.DB
	srv      *http.Server
}

func NewAdminServer(listener net.Listener, db *bbolt.DB) *AdminServer {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /ping", handler.Ping)
	mux.HandleFunc("GET /version", handler.Version)

	srv := &http.Server{
		Handler: mux,
	}

	return &AdminServer{
		listener: listener,
		db:       db,
		srv:      srv,
	}
}

func (s *AdminServer) Serve() error {
	slog.Info("admin server serving", "socket", s.listener.Addr())
	return s.srv.Serve(s.listener)
}

// No need for graceful shutdown in admin server
func (s *AdminServer) Close() error {
	slog.Info("admin server closing", "socket", s.listener.Addr())
	return s.srv.Close()
}
