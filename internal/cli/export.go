package cli

import (
	"fmt"
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
		cfg := config.LoadConfig()
		client, err := trac.NewTracClient(cfg.Trac.BaseURL, cfg.Trac.RPCPath)
		if err != nil {
			fmt.Println("❌ Failed to create Trac client:", err)
		}

		fmt.Println("Connected.")

		fmt.Println("Checking compability of Trac client...")

		if validateErr := client.ValidateExpectedMethods(); validateErr != nil {
			fmt.Println("❌ Trac client validation failed:", validateErr)
		}

		if validateVersionErr := client.ValidatePluginVersion(); validateVersionErr != nil {
			fmt.Println("❌ Trac plugin version validation failed:", validateVersionErr)
		}

		start := time.Now()

		if err := exporter.ExportTickets(client, "data", cfg.ExportOptions.IncludeClosedTickets, cfg.ExportOptions.IncludeAttachments); err != nil {
			fmt.Println("❌ Export failed:", err)
		}

		if err := exporter.ExportMilestones(client, "data"); err != nil {
			fmt.Println("❌ Export failed:", err)
		}

		if err := exporter.ExportWiki(client, "data", cfg.ExportOptions.IncludeAttachments); err != nil {
			fmt.Println("❌ Export failed:", err)
		}

		elapsed := time.Since(start)
		fmt.Printf("Export completed in %s\n", elapsed)
	},
}
