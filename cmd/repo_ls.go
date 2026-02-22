package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var repoLsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List all repos",
	RunE: func(cmd *cobra.Command, args []string) error {
		data := viper.GetString("data")

		cmd.SilenceUsage = true

		adminClient, err := initAdminClient(data)
		if err != nil {
			return err
		}

		repos, err := adminClient.ListRepos()
		if err != nil {
			return fmt.Errorf("cannot list repos: %w", err)
		}

		if len(repos) == 0 {
			fmt.Println("No repos exist so far. You might want to create one with `artf repo new`")
			return nil
		}

		renderReposTable(repos)
		return nil
	},
}

func init() {
	repoCmd.AddCommand(repoLsCmd)
}
