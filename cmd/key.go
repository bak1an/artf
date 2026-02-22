package cmd

import (
	"fmt"
	"strconv"

	"github.com/bak1an/artf/admin"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage keys",
}

func init() {
	rootCmd.AddCommand(keyCmd)
}

func renderKeysTable(keys []*admin.Key) {
	maxKeyNameLength := 0
	for _, key := range keys {
		if len(key.Name) > maxKeyNameLength {
			maxKeyNameLength = len(key.Name)
		}
	}
	keyNameWidth := maxKeyNameLength + 2
	columns := []table.Column{
		{Title: "ID", Width: 5},
		{Title: "Name", Width: keyNameWidth},
		{Title: "ReadOnly", Width: 8},
		{Title: "CreatedAt", Width: 25},
		{Title: "LastUsedAt", Width: 25},
	}
	rows := []table.Row{}
	for _, key := range keys {
		idStr := strconv.FormatUint(key.ID, 10)
		readOnlyStr := "yes"
		if !key.ReadOnly {
			readOnlyStr = "no"
		}
		lastUsedStr := "never"
		if key.LastUsedAt != nil {
			lastUsedStr = key.LastUsedAt.Format("2006-01-02 15:04:05")
		}

		createdAtStr := key.CreatedAt.Format("2006-01-02 15:04:05")
		rows = append(rows, table.Row{
			idStr,
			key.Name,
			readOnlyStr,
			createdAtStr,
			lastUsedStr,
		})
	}
	baseStyle := lipgloss.NewStyle()

	styles := table.DefaultStyles()
	styles.Selected = lipgloss.NewStyle() // reset selected style as this table is not interactive

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(len(keys)+1),
		table.WithStyles(styles),
	)
	t.Blur()

	fmt.Println(baseStyle.Render(t.View()))
}
