package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bak1an/artf/admin"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	repoEditKeepCount int
	repoEditKeepDays  int
)

var repoEditCmd = &cobra.Command{
	Use:     "edit [id or name]",
	Aliases: []string{"update"},
	Short:   "Edit repo retention settings",
	Example: `
# Edit repo interactively
artf repo edit

# Edit repo by id
artf repo edit 1 --keep-count 10 --keep-days 60

# Edit repo by name
artf repo edit my-repo --keep-count 5
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

		var selectedRepo *admin.Repo

		if len(args) == 1 {
			// Try parsing as ID first, then match by name
			id, err := strconv.ParseUint(args[0], 10, 64)
			if err == nil {
				for _, r := range existingRepos {
					if r.ID == id {
						selectedRepo = r
						break
					}
				}
			}
			if selectedRepo == nil {
				for _, r := range existingRepos {
					if r.Name == args[0] {
						selectedRepo = r
						break
					}
				}
			}
			if selectedRepo == nil {
				return fmt.Errorf("repo %q not found", args[0])
			}
		} else {
			options := make([]huh.Option[*admin.Repo], len(existingRepos))
			for i, repo := range existingRepos {
				options[i] = huh.Option[*admin.Repo]{
					Key:   fmt.Sprintf("%s (id: %d)", repo.Name, repo.ID),
					Value: repo,
				}
			}

			selectRepo := huh.NewSelect[*admin.Repo]().
				Title("Select repo to edit (type / to filter)").
				Options(options...).
				Value(&selectedRepo)

			err = selectRepo.Run()
			if err != nil {
				return err
			}
		}

		var keepCount *int
		var keepDays *int

		if cmd.Flags().Changed("keep-count") {
			keepCount = &repoEditKeepCount
		} else {
			keepCountStr := strconv.Itoa(selectedRepo.KeepCount)
			keepCountInput := huh.NewInput().
				Title("Keep count (0 = keep all) ").
				Placeholder("0").
				Value(&keepCountStr).
				Inline(true).
				Validate(func(s string) error {
					v, err := strconv.Atoi(strings.TrimSpace(s))
					if err != nil {
						return fmt.Errorf("keep count must be an integer")
					}
					if v < 0 {
						return fmt.Errorf("keep count must be >= 0")
					}
					return nil
				})
			err = keepCountInput.Run()
			if err != nil {
				return err
			}
			v, _ := strconv.Atoi(strings.TrimSpace(keepCountStr))
			keepCount = &v
		}

		if cmd.Flags().Changed("keep-days") {
			keepDays = &repoEditKeepDays
		} else {
			keepDaysStr := strconv.Itoa(selectedRepo.KeepDays)
			keepDaysInput := huh.NewInput().
				Title("Keep days (0 = keep all) ").
				Placeholder("0").
				Value(&keepDaysStr).
				Inline(true).
				Validate(func(s string) error {
					v, err := strconv.Atoi(strings.TrimSpace(s))
					if err != nil {
						return fmt.Errorf("keep days must be an integer")
					}
					if v < 0 {
						return fmt.Errorf("keep days must be >= 0")
					}
					return nil
				})
			err = keepDaysInput.Run()
			if err != nil {
				return err
			}
			v, _ := strconv.Atoi(strings.TrimSpace(keepDaysStr))
			keepDays = &v
		}

		repo, err := adminClient.UpdateRepo(selectedRepo.ID, keepCount, keepDays)
		if err != nil {
			return fmt.Errorf("cannot update repo: %w", err)
		}

		fmt.Printf("Repo '%s' updated (id: %d)\n", repo.Name, repo.ID)
		return nil
	},
}

func init() {
	repoCmd.AddCommand(repoEditCmd)
	repoEditCmd.Flags().IntVar(&repoEditKeepCount, "keep-count", 0, "number of artifacts to keep (0 = keep all)")
	repoEditCmd.Flags().IntVar(&repoEditKeepDays, "keep-days", 0, "number of days to keep artifacts (0 = keep all)")
}
