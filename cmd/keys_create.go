package cmd

import (
	"fmt"
	"strings"

	"github.com/bak1an/artf/admin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var keysCreateCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"new"},
	Short:   "Create a new key",
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

		var name string
		var readOnlyInput string
		fmt.Print("Enter key name: ")
		fmt.Scanln(&name)
		fmt.Print("Is this key read-only? (y/n): ")
		fmt.Scanln(&readOnlyInput)

		readOnly := strings.TrimSpace(strings.ToLower(readOnlyInput)) == "y"
		name = strings.TrimSpace(name)
		if name == "" {
			return fmt.Errorf("key name cannot be empty")
		}

		key, err := adminClient.CreateKey(name, readOnly)
		if err != nil {
			return fmt.Errorf("cannot create key: %w", err)
		}

		fmt.Printf("Key created: %s\n", *key.Key)

		return nil
	},
}

func init() {
	keysCmd.AddCommand(keysCreateCmd)
}
