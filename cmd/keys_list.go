package cmd

import (
	"fmt"

	"github.com/bak1an/artf/admin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var keysListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all keys",
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
		return nil
	},
}

func printKeysTable(keys []*admin.Key) {
	fmt.Println("+----+----------------------+----------+---------------------------+---------------------------+")
	fmt.Println("| ID | Name                 | ReadOnly | CreatedAt                 | LastUsedAt                |")
	fmt.Println("+----+----------------------+----------+---------------------------+---------------------------+")
	for _, key := range keys {
		var lastUsedStr string
		if key.LastUsedAt != nil {
			lastUsedStr = key.LastUsedAt.Format("2006-01-02 15:04:05")
		} else {
			lastUsedStr = "never"
		}
		fmt.Printf("| %-2d | %-20s | %-8t | %-25s | %-25s |\n",
			key.ID,
			key.Name,
			key.ReadOnly,
			key.CreatedAt.Format("2006-01-02 15:04:05"),
			lastUsedStr,
		)
	}
	fmt.Println("+----+----------------------+----------+---------------------------+---------------------------+")
}

func init() {
	keysCmd.AddCommand(keysListCmd)
}
