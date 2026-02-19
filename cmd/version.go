package cmd

import (
	"fmt"

	"github.com/bak1an/artf/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version",
	Run: func(cmd *cobra.Command, args []string) {
		vv := version.GetBuildInfo()
		fmt.Printf("artf %s (%s-%s)\n", vv.GitTag, vv.GitBranch, vv.GitRev)
		fmt.Printf("build on %s with go %s\n", vv.BuildTime, vv.GoVersion)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
