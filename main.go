package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/bak1an/artf/handler"
	buildInfo "github.com/bak1an/artf/version"
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

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", handler.Ping)
	mux.HandleFunc("/version", handler.Version)
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", *host, *port),
		Handler: mux,
	}
	slog.Info("starting server", "address", server.Addr)

	if err := server.ListenAndServe(); err != nil {
		slog.Error("cannot start server", "error", err)
		return
	}
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
