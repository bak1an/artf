package cmd

import (
	"fmt"
	"strconv"

	"github.com/bak1an/artf/admin"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var repoRmYes bool

var repoRmCmd = &cobra.Command{
	Use:     "rm [id]",
	Aliases: []string{"delete"},
	Short:   "Delete a repo",
	Example: `
# Delete repo interactively
artf repo rm

# Delete repo by id
artf repo rm 1

# Delete repo by id skipping confirmation
artf repo rm 1 --yes
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := viper.GetString("data")

		cmd.SilenceUsage = true

		adminClient, err := initAdminClient(data)
		if err != nil {
			return err
		}

		existingRepos, err := adminClient.ListRepos()
		if err != nil {
			return err
		}

		if len(existingRepos) == 0 {
			fmt.Println("No repos exist so far. You might want to create one with `artf repo new`")
			return nil
		}

		var toDelete *admin.Repo

		if len(args) == 1 {
			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid repo id %q: %w", args[0], err)
			}
			for _, r := range existingRepos {
				if r.ID == id {
					toDelete = r
					break
				}
			}
			if toDelete == nil {
				return fmt.Errorf("repo with id %d not found", id)
			}
		} else {
			options := make([]huh.Option[*admin.Repo], len(existingRepos))
			for i, repo := range existingRepos {
				options[i] = huh.Option[*admin.Repo]{
					Key:   repo.Name,
					Value: repo,
				}
			}
			toDeleteInput := huh.NewSelect[*admin.Repo]().
				Title("Select repo to delete (type / to filter)").
				Options(options...).
				Value(&toDelete)

			err = toDeleteInput.Run()
			if err != nil {
				return err
			}
		}

		confirm := repoRmYes
		if !confirm {
			confirmInput := huh.NewConfirm().
				Title(fmt.Sprintf("Are you sure you want to delete repo '%s'?", toDelete.Name)).
				Value(&confirm).
				Inline(true)
			err = confirmInput.Run()
			if err != nil {
				return err
			}
		}

		if !confirm {
			fmt.Println("Ok, doing nothing")
			return nil
		}

		err = adminClient.DeleteRepo(toDelete.ID)
		if err != nil {
			return fmt.Errorf("cannot delete repo: %w", err)
		}
		fmt.Println("Repo deleted")

		return nil
	},
}

func init() {
	repoCmd.AddCommand(repoRmCmd)
	repoRmCmd.Flags().BoolVar(&repoRmYes, "yes", false, "skip confirmation")
}
