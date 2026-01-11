package cmd

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/bak1an/artf/admin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.etcd.io/bbolt"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start the artifact server",
	RunE: func(cmd *cobra.Command, args []string) error {
		data := viper.GetString("data")
		slog.Debug("serving data directory", "data", data)

		db, err := openDatabase(data)
		if err != nil {
			return fmt.Errorf("cannot open database: %w", err)
		}
		defer db.Close()

		adminServer, err := admin.NewAdminServer(data, db)
		if err != nil {
			return fmt.Errorf("cannot initialize admin server: %w", err)
		}

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

		adminErr := make(chan error, 1)

		go func() {
			if err := adminServer.Serve(); err != nil && err != http.ErrServerClosed {
				slog.Error("cannot run admin server, crashing", "error", err)
				adminErr <- err
			}
		}()

		select {
		case <-stop:
			slog.Info("shutting down")
		case err := <-adminErr:
			slog.Error("failed to run admin server, crashing", "error", err)
			return err
		}

		err = adminServer.Close()
		if err != nil {
			slog.Error("failed to close admin server", "error", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func openDatabase(data string) (*bbolt.DB, error) {
	dbPath := filepath.Join(data, dbFilename)
	slog.Info("opening database", "path", dbPath)
	db, err := bbolt.Open(dbPath, dbFileMode, &bbolt.Options{Timeout: dbOpenTimeout})
	if err != nil {
		return nil, fmt.Errorf("cannot open database: %w", err)
	}
	return db, nil
}
