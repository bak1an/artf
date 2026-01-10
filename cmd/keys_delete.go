package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var keysDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a key",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("delete")
	},
}

func init() {
	keysCmd.AddCommand(keysDeleteCmd)
}
