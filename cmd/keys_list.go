package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var keysListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		data := viper.GetString("data")

		cmd.SilenceUsage = true

		adminClient, err := initAdminClient(data)
		if err != nil {
			return err
		}

		keys, err := adminClient.ListKeys()
		if err != nil {
			return fmt.Errorf("cannot list keys: %w", err)
		}

		if len(keys) == 0 {
			fmt.Println("No keys exist so far. You might want to create one with `artf keys create`")
			return nil
		}

		renderKeysTable(keys)
		return nil
	},
}

func init() {
	keysCmd.AddCommand(keysListCmd)
}
