package cli

import (
	"fmt"
	"trac2gitlab/internal/config"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version of trac2gitlab",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("trac2gitlab version %s\n", config.AppVersion)
	},
}
