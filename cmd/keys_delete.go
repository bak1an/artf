package cmd

import (
	"fmt"

	"github.com/bak1an/artf/admin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var keysDeleteCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"rm"},
	Short:   "Delete a key",
	RunE: func(cmd *cobra.Command, args []string) error {
		data := viper.GetString("data")
		adminClient, err := admin.NewAdminClient(data)
		if err != nil {
			return fmt.Errorf("cannot create admin client: %w", err)
		}

		err = adminClient.Ping()
		if err != nil {
			return fmt.Errorf("cannot ping admin server: %w", err)
		}

		keys, err := adminClient.ListKeys()
		if err != nil {
			return fmt.Errorf("cannot list keys: %w", err)
		}

		if len(keys) == 0 {
			fmt.Println("No keys exist so far. You might want to create one with `artf keys create`")
			return nil
		}

		printKeysTable(keys)
		fmt.Print("Enter key ID to delete: ")

		var id uint64
		fmt.Scanln(&id)

		if id == 0 {
			return fmt.Errorf("invalid key ID")
		}

		err = adminClient.DeleteKey(id)
		if err != nil {
			return fmt.Errorf("cannot delete key: %w", err)
		}
		fmt.Println("Key deleted")

		return nil
	},
}

func init() {
	keysCmd.AddCommand(keysDeleteCmd)
}
