package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var keysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all keys",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("list")
	},
}

func init() {
	keysCmd.AddCommand(keysListCmd)
}
