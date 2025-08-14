package cli

import (
	"log/slog"

	"github.com/bnidev/trac2gitlab/internal/config"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version of trac2gitlab",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("trac2gitlab", "version", config.AppVersion)
	},
}
