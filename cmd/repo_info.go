package cmd

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/bak1an/artf/admin"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var repoInfoCmd = &cobra.Command{
	Use:     "info [id]",
	Aliases: []string{"show"},
	Short:   "Show detailed info about a repo",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := viper.GetString("data")

		cmd.SilenceUsage = true

		adminClient, err := initAdminClient(data)
		if err != nil {
			return err
		}

		existingRepos, err := adminClient.ListRepos()
		if err != nil {
			return fmt.Errorf("cannot list repos: %w", err)
		}

		if len(existingRepos) == 0 {
			fmt.Println("No repos exist so far. You might want to create one with `artf repo new`")
			return nil
		}

		var selectedRepoID uint64
		if len(args) == 1 {
			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid repo id %q: %w", args[0], err)
			}
			found := false
			for _, repo := range existingRepos {
				if repo.ID == id {
					selectedRepoID = id
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("repo with id %d not found", id)
			}
		} else {
			var selectedRepo *admin.Repo
			options := make([]huh.Option[*admin.Repo], len(existingRepos))
			for i, repo := range existingRepos {
				options[i] = huh.Option[*admin.Repo]{
					Key:   fmt.Sprintf("%s (id: %d)", repo.Name, repo.ID),
					Value: repo,
				}
			}

			selectRepo := huh.NewSelect[*admin.Repo]().
				Title("Select repo to inspect (type / to filter)").
				Options(options...).
				Value(&selectedRepo)

			err = selectRepo.Run()
			if err != nil {
				return err
			}
			selectedRepoID = selectedRepo.ID
		}

		info, err := adminClient.GetRepoInfo(selectedRepoID)
		if err != nil {
			return fmt.Errorf("cannot get repo info: %w", err)
		}

		renderRepoInfo(info, data)
		return nil
	},
}

func init() {
	repoCmd.AddCommand(repoInfoCmd)
}

func renderRepoInfo(info *admin.RepoInfoResponse, dataDir string) {
	fmt.Println("Repository")
	fmt.Printf("  ID:            %d\n", info.Repo.ID)
	fmt.Printf("  Name:          %s\n", info.Repo.Name)
	fmt.Printf("  Type:          %s\n", info.Repo.Type)
	fmt.Printf("  Path:          %s\n", displayAbsPath(dataDir, info.Repo.Path))
	fmt.Printf("  KeepCount:     %d\n", info.Repo.KeepCount)
	fmt.Printf("  KeepDays:      %d\n", info.Repo.KeepDays)
	fmt.Printf("  CreatedAt:     %s\n", info.Repo.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  UpdatedAt:     %s\n", info.Repo.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  ArtifactCount: %d\n", info.Repo.ArtifactCount)
	fmt.Println()
	fmt.Println("Artifacts")

	if len(info.Artifacts) == 0 {
		fmt.Println("No artifacts yet")
		return
	}

	slices.SortFunc(info.Artifacts, func(a, b *admin.ArtifactInfo) int {
		if d := b.CreatedAt.Compare(a.CreatedAt); d != 0 {
			return d
		}
		return cmp.Compare(b.ID, a.ID)
	})

	rows := make([]table.Row, len(info.Artifacts))
	maxNameWidth := len("Name")
	maxPathWidth := len("Path")
	for i, artifact := range info.Artifacts {
		if len(artifact.Name) > maxNameWidth {
			maxNameWidth = len(artifact.Name)
		}
		path := displayAbsPath(dataDir, artifact.Path)
		if len(path) > maxPathWidth {
			maxPathWidth = len(path)
		}

		rows[i] = table.Row{
			strconv.FormatUint(artifact.ID, 10),
			artifact.Name,
			strconv.FormatUint(artifact.RepoID, 10),
			path,
			artifact.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	columns := []table.Column{
		{Title: "ID", Width: 5},
		{Title: "Name", Width: maxNameWidth + 2},
		{Title: "RepoID", Width: 8},
		{Title: "Path", Width: maxPathWidth + 2},
		{Title: "CreatedAt", Width: len(time.Now().Format("2006-01-02 15:04:05")) + 2},
	}

	styles := table.DefaultStyles()
	styles.Selected = lipgloss.NewStyle()

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(len(rows)+1),
		table.WithStyles(styles),
	)
	t.Blur()

	fmt.Println(lipgloss.NewStyle().Render(t.View()))
}

func displayAbsPath(dataDir, p string) string {
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}

	abs, err := filepath.Abs(filepath.Join(dataDir, p))
	if err != nil {
		return filepath.Clean(filepath.Join(dataDir, p))
	}
	return abs
}
