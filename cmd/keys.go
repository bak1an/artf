package cmd

import (
	"github.com/spf13/cobra"
)

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage keys",
}

func init() {
	rootCmd.AddCommand(keysCmd)
}
