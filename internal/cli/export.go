package cli

import (
	"log/slog"
	"time"
	"trac2gitlab/internal/config"
	"trac2gitlab/internal/exporter"
	"trac2gitlab/pkg/trac"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export tickets, wiki, users, and attachments from Trac",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			slog.Error("Failed to load configuration", "errorMsg", err)
			return
		}

		client, err := trac.NewTracClient(cfg.Trac.BaseURL, cfg.Trac.RPCPath)
		if err != nil {
			slog.Error("Failed to create Trac client", "errorMsg", err)
			return
		}

		slog.Debug("Checking compatibility of Trac client...")

		if validateErr := client.ValidateExpectedMethods(); validateErr != nil {
			slog.Error("Trac client validation failed", "errorMsg", validateErr)
			return
		}

		if validateVersionErr := client.ValidatePluginVersion(); validateVersionErr != nil {
			slog.Error("Trac plugin version validation failed", "errorMsg", validateVersionErr)
			return
		}

		start := time.Now()

		if err := exporter.ExportTickets(client, "data", cfg.ExportOptions.IncludeClosedTickets, cfg.ExportOptions.IncludeAttachments); err != nil {
			slog.Error("Ticket export failed", "errorMsg", err)
		}

		if err := exporter.ExportMilestones(client, "data"); err != nil {
			slog.Error("Milestone export failed", "errorMsg", err)
		}

		if cfg.ExportOptions.IncludeWiki {
			if err := exporter.ExportWiki(client, "data", cfg.ExportOptions.IncludeAttachments); err != nil {
				slog.Error("Wiki export failed", "errorMsg", err)
			}
		}

		if cfg.ExportOptions.IncludeUsers {
			if err := exporter.ExportUsers(client, "data"); err != nil {
				slog.Error("User export failed", "errorMsg", err)
			}
		}

		elapsed := time.Since(start)
		slog.Info("Export completed successfully", "duration", elapsed)
	},
}
