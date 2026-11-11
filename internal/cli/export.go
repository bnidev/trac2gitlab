package cli

import (
	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/bnidev/trac2gitlab/internal/pipeline"
	"github.com/spf13/cobra"
)

func exportCmd(ctx *app.AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Export tickets, wiki, users, and attachments from Trac",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := *ctx.Config
			return pipeline.Export(&cfg, nil)
		},
	}
}
