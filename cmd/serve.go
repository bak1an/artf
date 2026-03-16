package cmd

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

	"github.com/bak1an/artf/admin"
	"github.com/bak1an/artf/server"
	"github.com/bak1an/artf/store/bblt"
	"github.com/coreos/go-systemd/v22/activation"
	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.etcd.io/bbolt"
)

const (
	shutdownTimeout = 10 * time.Second
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start the artifact server",
	RunE: func(cmd *cobra.Command, args []string) error {
		data := viper.GetString("data")
		slog.Debug("serving data directory", "data", data)

		cmd.SilenceUsage = true

		maxUploadSize, err := resolveMaxUploadSize(viper.GetViper())
		if err != nil {
			return fmt.Errorf("invalid max upload size: %w", err)
		}

		bboltDB, err := openDatabase(data)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}

		db, err := bblt.NewBboltStore(bboltDB)
		if err != nil {
			if err := bboltDB.Close(); err != nil {
				slog.Error("failed to close database after error", "error", err)
			}
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		defer db.Close()

		adminServer, err := admin.NewAdminServer(data, db)
		if err != nil {
			return fmt.Errorf("failed to initialize admin server: %w", err)
		}

		listener, err := getListener(cmd)
		if err != nil {
			return fmt.Errorf("failed to get listener: %w", err)
		}

		srv, err := server.NewServer(data, listener, db, maxUploadSize)
		if err != nil {
			return fmt.Errorf("failed to initialize server: %w", err)
		}

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

		adminErr := make(chan error, 1)
		serverErr := make(chan error, 1)

		go func() {
			if err := adminServer.Serve(); err != nil && err != http.ErrServerClosed {
				adminErr <- err
			}
		}()

		go func() {
			if err := srv.Serve(); err != nil && err != http.ErrServerClosed {
				serverErr <- err
			}
		}()

		daemon.SdNotify(false, daemon.SdNotifyReady)

		select {
		case <-stop:
			slog.Info("shutting down")
		case err := <-adminErr:
			slog.Error("failed to run admin server, crashing", "error", err)
			return err
		case err := <-serverErr:
			slog.Error("failed to run server, crashing", "error", err)
			return err
		}

		daemon.SdNotify(false, daemon.SdNotifyStopping)

		err = adminServer.Close()
		if err != nil {
			slog.Error("failed to close admin server", "error", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		err = srv.Shutdown(ctx)
		if err != nil {
			slog.Error("failed to shutdown server", "error", err)
			return err
		}

		slog.Info("all down, bye!")

		return nil
	},
}

func init() {
	serveCmd.Flags().Bool("systemd", false, "use systemd socket activation (replaces host and port flags)")
	serveCmd.Flags().StringP("host", "H", "127.0.0.1", "host to listen on (ignored if systemd is used)")
	serveCmd.Flags().IntP("port", "P", 8365, "port to listen on (ignored if systemd is used)")
	serveCmd.Flags().String(maxUploadSizeKey, defaultMaxUploadSize, "maximum upload size in bytes or with k, m, g suffixes")

	if err := viper.BindEnv(maxUploadSizeKey, "ARTF_MAX_UPLOAD_SIZE"); err != nil {
		panic(err)
	}

	rootCmd.AddCommand(serveCmd)
}

func getListener(cmd *cobra.Command) (net.Listener, error) {
	if systemd, err := cmd.Flags().GetBool("systemd"); err != nil {
		return nil, err
	} else if systemd {
		slog.Info("using systemd socket activation")
		return systemdListener()
	}
	host, err := cmd.Flags().GetString("host")
	if err != nil {
		return nil, err
	}
	port, err := cmd.Flags().GetInt("port")
	if err != nil {
		return nil, err
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	slog.Info("listening on TCP address", "address", addr)
	return net.Listen("tcp", addr)
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

func openDatabase(data string) (*bbolt.DB, error) {
	dbPath := filepath.Join(data, dbFilename)
	slog.Info("opening database", "path", dbPath)
	db, err := bbolt.Open(dbPath, dbFileMode, &bbolt.Options{Timeout: dbOpenTimeout})
	if err != nil {
		return nil, fmt.Errorf("cannot open database: %w", err)
	}
	return db, nil
}
