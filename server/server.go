package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"github.com/bak1an/artf/server/handler"
	"github.com/bak1an/artf/store"
)

type Server struct {
	listener net.Listener
	dataDir  string
	db       store.Store
	srv      *http.Server
}

func NewServer(dataDir string, listener net.Listener, db store.Store) (*Server, error) {
	baseLogger := slog.Default().With("server", "api")

	mux := createMux()

	h := RecoverMiddleware( // recover panics
		RequestID( // add request id
			RequestContextLogger(baseLogger)( // add logger to the context
				LoggingMiddleware(mux), // log response times via above logger
			),
		),
	)

	srv := &http.Server{
		Handler: h,
	}

	return &Server{dataDir: dataDir, listener: listener, db: db, srv: srv}, nil
}

func createMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /ping", handler.Ping)
	mux.HandleFunc("GET /version", handler.Version)

	return mux
}

func (s *Server) Serve() error {
	slog.Info("serving", "address", s.listener.Addr())
	return s.srv.Serve(s.listener)
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down", "address", s.listener.Addr())
	return s.srv.Shutdown(ctx)
}
