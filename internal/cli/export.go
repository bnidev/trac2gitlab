package cli

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/bnidev/trac2gitlab/internal/exporter"
	"github.com/bnidev/trac2gitlab/pkg/trac"

	"github.com/spf13/cobra"
)

func exportCmd(ctx *app.AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Export tickets, wiki, users, and attachments from Trac",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := *ctx.Config

			client, err := trac.NewTracClient(&cfg)
			if err != nil {
				return fmt.Errorf("create Trac client: %w", err)
			}
			if err := client.ValidateExpectedMethods(); err != nil {
				return fmt.Errorf("trac client validation: %w", err)
			}
			if err := client.ValidatePluginVersion(); err != nil {
				return fmt.Errorf("trac plugin version validation: %w", err)
			}

			start := time.Now()
			steps := []struct {
				name    string
				enabled bool
				fn      func() error
			}{
				{"ticket fields", cfg.ExportOptions.IncludeTicketFields, func() error { return exporter.ExportTicketFields(client, &cfg) }},
				{"tickets", true, func() error { return exporter.ExportTickets(client, &cfg) }},
				{"milestones", true, func() error { return exporter.ExportMilestones(client, &cfg) }},
				{"wiki", cfg.ExportOptions.IncludeWiki, func() error { return exporter.ExportWiki(client, &cfg) }},
				{"users", cfg.ExportOptions.IncludeUsers, func() error { return exporter.ExportUsers(client, &cfg) }},
			}
			for _, s := range steps {
				if !s.enabled {
					continue
				}
				if err := s.fn(); err != nil {
					return fmt.Errorf("export %s: %w", s.name, err)
				}
			}
			slog.Info("Export completed successfully", "duration", time.Since(start))
			return nil
		},
	}
}
