package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var keysCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new key",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("create")
	},
}

func init() {
	keysCmd.AddCommand(keysCreateCmd)
}
