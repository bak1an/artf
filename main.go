package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/bak1an/artf/handler"
	buildInfo "github.com/bak1an/artf/version"
	"github.com/coreos/go-systemd/v22/activation"
	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/spf13/pflag"
	bolt "go.etcd.io/bbolt"
)

const (
	dbFilename    = "artf.db"
	dbFileMode    = 0600
	dbOpenTimeout = 5 * time.Second
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelInfo)

	version := pflag.BoolP("version", "v", false, "print version and exit")
	dataDir := pflag.StringP("data", "d", "", "data directory, must exist and be writable")
	port := pflag.IntP("port", "p", 8365, "port to listen on")
	host := pflag.StringP("host", "h", "127.0.0.1", "host to listen on")
	systemd := pflag.BoolP("systemd", "s", false, "get listen socket from systemd")

	pflag.Parse()

	if *version {
		vv := buildInfo.GetBuildInfo()
		fmt.Printf("artf %s (%s-%s)\n", vv.GitTag, vv.GitBranch, vv.GitRev)
		fmt.Printf("build on %s with go %s\n", vv.BuildTime, vv.GoVersion)
		os.Exit(0)
	}

	if *dataDir == "" {
		slog.Error("data directory is required")
		pflag.PrintDefaults()
		os.Exit(1)
	}

	if err := checkDirWritable(*dataDir); err != nil {
		slog.Error("invalid data directory", "error", err)
		os.Exit(1)
	}

	db, err := bolt.Open(filepath.Join(*dataDir, dbFilename), dbFileMode, &bolt.Options{Timeout: dbOpenTimeout})
	if err != nil {
		slog.Error("cannot open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	var listener net.Listener

	if *systemd {
		listener, err = systemdListener()
		if err != nil {
			slog.Error("cannot get listen socket from systemd", "error", err)
			os.Exit(1)
		}
	} else {
		addr := fmt.Sprintf("%s:%d", *host, *port)
		slog.Info("listening on address", "address", addr)
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			slog.Error("cannot listen on address", "error", err)
			os.Exit(1)
		}
		slog.Info("listening on address", "address", addr)
	}

	server := &http.Server{
		Handler: createMux(db),
	}
	slog.Info("starting server", "address", server.Addr)

	// Setup graceful shutdown on SIGINT or SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	daemon.SdNotify(false, daemon.SdNotifyReady)

	// Wait for either SIGINT or server error
	select {
	case err := <-serverErr:
		slog.Error("cannot start server", "error", err)
		return
	case sig := <-sigChan:
		slog.Info("received signal, shutting down gracefully", "signal", sig)

		daemon.SdNotify(false, daemon.SdNotifyStopping)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			slog.Error("server shutdown error", "error", err)
			return
		}
		slog.Info("server shutdown complete")
	}
}

func createMux(db *bolt.DB) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", handler.Ping)
	mux.HandleFunc("/version", handler.Version)
	return mux
}

func checkDirWritable(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", dir)
		}
		return fmt.Errorf("cannot access directory: %v", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", dir)
	}

	testFile := filepath.Join(dir, ".write_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("directory is not writable: %v", err)
	}

	err = os.Remove(testFile)
	if err != nil {
		return fmt.Errorf("cannot remove test file: %v", err)
	}

	return nil
}

func systemdListener() (net.Listener, error) {
	listeners, err := activation.Listeners()

	if err != nil {
		return nil, err
	}

	if len(listeners) == 0 {
		return nil, fmt.Errorf("no listeners provided by systemd")
	}

	if len(listeners) > 1 {
		return nil, fmt.Errorf("multiple listeners provided by systemd, only one is supported")
	}

	return listeners[0], nil
}
