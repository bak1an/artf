package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	dbFilename    = "artf0.db"
	dbFileMode    = 0600
	dbOpenTimeout = 5 * time.Second
)

var rootCmd = &cobra.Command{
	Use:              "artf",
	Short:            "artf is a tool for managing artifacts",
	TraverseChildren: true,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initializeConfig(cmd)
	},
}

func initializeConfig(cmd *cobra.Command) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	defaultData := filepath.Join(home, ".artf")
	viper.SetDefault("data", defaultData)

	viper.SetEnvPrefix("ARTF")
	viper.BindEnv("data")

	err = viper.BindPFlags(cmd.Flags())
	if err != nil {
		return err
	}

	verbose := viper.GetBool("verbose")
	if verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}

	data := viper.GetString("data")
	if data == "" {
		return fmt.Errorf("data directory is required")
	}

	if data == defaultData {
		if _, err := os.Stat(data); os.IsNotExist(err) {
			slog.Info("creating default data directory", "directory", data)
			if err := os.MkdirAll(data, 0770); err != nil {
				return fmt.Errorf("cannot create data directory: %v", err)
			}
		}
	}

	if err = checkDirWritable(data); err != nil {
		return fmt.Errorf("data directory is not writable: %v", err)
	}

	data, err = filepath.Abs(data)
	if err != nil {
		return fmt.Errorf("cannot get absolute path of data directory: %v", err)
	}

	viper.Set("data", data)

	return nil
}

func init() {
	rootCmd.PersistentFlags().StringP("data", "d", "", "data directory (default: $HOME/.artf)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
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

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
