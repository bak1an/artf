package cmd

import (
	"fmt"
	"log/slog"

	"github.com/bak1an/artf/admin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of the server",
	RunE: func(cmd *cobra.Command, args []string) error {
		data := viper.GetString("data")
		adminClient, err := admin.NewAdminClient(data)
		if err != nil {
			return fmt.Errorf("cannot create admin client: %w", err)
		}
		slog.Info("Checking status of admin server", "path", adminClient.Path())

		err = adminClient.Ping()
		if err != nil {
			return fmt.Errorf("cannot ping admin server: %w", err)
		}

		v, err := adminClient.Version()
		if err != nil {
			return fmt.Errorf("cannot get running version: %w", err)
		}

		fmt.Printf("Admin server is running at %s\n", adminClient.Path())
		fmt.Printf("Running version: %s\n", v.GitTag)
		fmt.Printf("Build time: %s\n", v.BuildTime)
		fmt.Printf("Git branch: %s\n", v.GitBranch)
		fmt.Printf("Git revision: %s\n", v.GitRev)
		fmt.Printf("Go version: %s\n", v.GoVersion)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
