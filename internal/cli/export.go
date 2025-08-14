package cli

import (
	"log/slog"
	"time"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/exporter"
	"github.com/bnidev/trac2gitlab/pkg/trac"

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

		client, err := trac.NewTracClient(&cfg)
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

		if cfg.ExportOptions.IncludeTicketFields {
			if err := exporter.ExportTicketFields(client, &cfg); err != nil {
				slog.Error("Ticket fields export failed", "errorMsg", err)
			}
		}

		if err := exporter.ExportTickets(client, &cfg); err != nil {
			slog.Error("Ticket export failed", "errorMsg", err)
		}

		if err := exporter.ExportMilestones(client, &cfg); err != nil {
			slog.Error("Milestone export failed", "errorMsg", err)
		}

		if cfg.ExportOptions.IncludeWiki {
			if err := exporter.ExportWiki(client, &cfg); err != nil {
				slog.Error("Wiki export failed", "errorMsg", err)
			}
		}

		if cfg.ExportOptions.IncludeUsers {
			if err := exporter.ExportUsers(client, &cfg); err != nil {
				slog.Error("User export failed", "errorMsg", err)
			}
		}

		elapsed := time.Since(start)
		slog.Info("Export completed successfully", "duration", elapsed)
	},
}
