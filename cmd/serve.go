package cmd

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"

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
			return err
		}
		defer db.Close()

		adminServer, err := prepareAdminServer(data, db)
		if err != nil {
			return fmt.Errorf("cannot initialize admin server: %v", err)
		}

		return adminServer.Serve()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func openDatabase(data string) (*bbolt.DB, error) {
	db, err := bbolt.Open(filepath.Join(data, dbFilename), dbFileMode, &bbolt.Options{Timeout: dbOpenTimeout})
	if err != nil {
		return nil, fmt.Errorf("cannot open database: %v", err)
	}
	return db, nil
}

func prepareAdminServer(data string, db *bbolt.DB) (*admin.AdminServer, error) {
	socketPath := filepath.Join(data, controlSocket)

	if _, err := os.Stat(socketPath); err == nil {
		slog.Debug("removing existing socket file", "socket", socketPath)
		err := os.Remove(socketPath)
		if err != nil {
			return nil, fmt.Errorf("cannot remove socket file: %v", err)
		}
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("cannot listen on socket: %v", err)
	}

	if err := os.Chmod(socketPath, controlSocketMode); err != nil {
		return nil, fmt.Errorf("cannot change socket file mode: %v", err)
	}

	return admin.NewAdminServer(listener, db), nil
}
