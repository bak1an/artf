package cmd

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/bak1an/artf/internal/validate"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	repoNewName      string
	repoNewKeepCount int
	repoNewKeepDays  int
)

var repoNewCmd = &cobra.Command{
	Use:     "new",
	Aliases: []string{"create"},
	Short:   "Create a new repo",
	Example: `
# Create new repo interactively
artf repo new

# Create new repo non-interactively
artf repo new --name myrepo --keep-count 5 --keep-days 30
`,
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
		existingRepoNames := make([]string, len(existingRepos))
		for i, repo := range existingRepos {
			existingRepoNames[i] = repo.Name
		}

		validateName := func(s string) error {
			s = strings.TrimSpace(s)
			if err := validate.RepoName(s); err != nil {
				return err
			}
			if slices.Contains(existingRepoNames, s) {
				return fmt.Errorf("repo name must be unique, %q already exists", s)
			}
			return nil
		}

		var name string
		keepCount := repoNewKeepCount
		keepDays := repoNewKeepDays

		if cmd.Flags().Changed("name") {
			name = strings.TrimSpace(repoNewName)
			if err := validateName(name); err != nil {
				return err
			}
		} else {
			nameInput := huh.NewInput().
				Title("Repo name ").
				Placeholder("my-repo").
				Value(&name).
				CharLimit(validate.MaxRepoNameLength).
				Inline(true).
				Validate(validateName)
			err = nameInput.Run()
			if err != nil {
				return err
			}
			name = strings.TrimSpace(name)
		}

		if !cmd.Flags().Changed("keep-count") {
			keepCountStr := "0"
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
			keepCount, _ = strconv.Atoi(strings.TrimSpace(keepCountStr))
		}

		if !cmd.Flags().Changed("keep-days") {
			keepDaysStr := "0"
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
			keepDays, _ = strconv.Atoi(strings.TrimSpace(keepDaysStr))
		}

		repo, err := adminClient.CreateRepo(name, keepCount, keepDays)
		if err != nil {
			return fmt.Errorf("cannot create repo: %w", err)
		}

		fmt.Printf("Repo '%s' created (id: %d)\n", repo.Name, repo.ID)
		return nil
	},
}

func init() {
	repoCmd.AddCommand(repoNewCmd)
	repoNewCmd.Flags().StringVarP(&repoNewName, "name", "n", "", "repo name")
	repoNewCmd.Flags().IntVar(&repoNewKeepCount, "keep-count", 0, "number of artifacts to keep (0 = keep all)")
	repoNewCmd.Flags().IntVar(&repoNewKeepDays, "keep-days", 0, "number of days to keep artifacts (0 = keep all)")
}
