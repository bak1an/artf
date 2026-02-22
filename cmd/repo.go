package cmd

import (
	"fmt"
	"strconv"

	"github.com/bak1an/artf/admin"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repos",
}

func init() {
	rootCmd.AddCommand(repoCmd)
}

func renderReposTable(repos []*admin.Repo) {
	maxRepoNameLength := 0
	for _, repo := range repos {
		if len(repo.Name) > maxRepoNameLength {
			maxRepoNameLength = len(repo.Name)
		}
	}
	repoNameWidth := maxRepoNameLength + 2
	columns := []table.Column{
		{Title: "ID", Width: 5},
		{Title: "Name", Width: repoNameWidth},
		{Title: "Retention", Width: 10},
		{Title: "Artifacts", Width: 10},
		{Title: "CreatedAt", Width: 25},
	}
	rows := []table.Row{}
	for _, repo := range repos {
		idStr := strconv.FormatUint(repo.ID, 10)
		createdAtStr := repo.CreatedAt.Format("2006-01-02 15:04:05")
		retentionStr := ""
		if repo.KeepCount > 0 {
			retentionStr += fmt.Sprintf("%d", repo.KeepCount)
		}
		if repo.KeepDays > 0 {
			if retentionStr != "" {
				retentionStr += " / "
			}
			retentionStr += fmt.Sprintf("%dd", repo.KeepDays)
		}
		if retentionStr == "" {
			retentionStr = "all"
		}
		rows = append(rows, table.Row{
			idStr,
			repo.Name,
			retentionStr,
			strconv.Itoa(repo.ArtifactCount),
			createdAtStr,
		})
	}
	baseStyle := lipgloss.NewStyle()

	styles := table.DefaultStyles()
	styles.Selected = lipgloss.NewStyle()

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(len(repos)+1),
		table.WithStyles(styles),
	)
	t.Blur()

	fmt.Println(baseStyle.Render(t.View()))
}
