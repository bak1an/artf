package admin

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bak1an/artf/server"
	"github.com/bak1an/artf/server/handler"
	"github.com/bak1an/artf/store"
)

const (
	controlSocket     = "artf.sock"
	controlSocketMode = 0660
)

type AdminServer struct {
	socketPath string
	listener   net.Listener
	db         store.Store
	srv        *http.Server
}

func NewAdminServer(dataDir string, db store.Store) (*AdminServer, error) {
	socketPath := filepath.Join(dataDir, controlSocket)

	if err := cleanSocket(socketPath); err != nil {
		return nil, fmt.Errorf("cannot remove existing socket: %w", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("cannot listen on socket: %w", err)
	}

	if err := os.Chmod(socketPath, controlSocketMode); err != nil {
		return nil, fmt.Errorf("cannot change socket file mode: %w", err)
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /ping", handler.Ping)
	mux.HandleFunc("GET /version", handler.Version)

	mux.HandleFunc("GET /keys", listKeysHandler(db.APIKeys()))          // list all API keys
	mux.HandleFunc("PUT /keys", createKeyHandler(db.APIKeys()))         // create a new API key
	mux.HandleFunc("DELETE /keys/{id}", deleteKeyHandler(db.APIKeys())) // delete a key

	mux.HandleFunc("GET /repos", listReposHandler(db.Repos(), db.Artifacts()))                   // list all repos
	mux.HandleFunc("GET /repos/{id}", getRepoInfoHandler(db.Repos(), db.Artifacts()))            // get repo details
	mux.HandleFunc("PUT /repos", createRepoHandler(db.Repos(), dataDir))                         // create a new repo
	mux.HandleFunc("PATCH /repos/{id}", updateRepoHandler(db.Repos()))                           // update repo settings
	mux.HandleFunc("DELETE /repos/{id}", deleteRepoHandler(db.Repos(), db.Artifacts(), dataDir)) // delete a repo

	baseLogger := slog.Default().With("server", "admin")

	h := server.RecoverMiddleware( // recover panics
		server.RequestID( // add request id
			server.RequestContextLogger(baseLogger)( // add logger to the context
				server.LoggingMiddleware(mux), // log response times via above logger
			),
		),
	)

	srv := &http.Server{
		Handler: h,
	}

	return &AdminServer{
		socketPath: socketPath,
		listener:   listener,
		db:         db,
		srv:        srv,
	}, nil
}

func (s *AdminServer) Serve() error {
	slog.Info("admin server serving", "socket", s.listener.Addr())
	return s.srv.Serve(s.listener)
}

// No need for graceful shutdown in admin server
func (s *AdminServer) Close() error {
	slog.Info("admin server closing", "socket", s.listener.Addr())
	err := s.srv.Close()
	if err != nil {
		return fmt.Errorf("failed to close admin server: %w", err)
	}
	err = cleanSocket(s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to clean socket: %w", err)
	}
	return nil
}

func cleanSocket(socketPath string) error {
	if _, err := os.Stat(socketPath); err == nil || !os.IsNotExist(err) {
		slog.Debug("removing existing socket file", "socket", socketPath)
		err := os.Remove(socketPath)
		if err != nil {
			return fmt.Errorf("cannot remove socket file: %w", err)
		}
	}
	return nil
}
