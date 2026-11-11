package cli

import (
	"fmt"

	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/bnidev/trac2gitlab/internal/pipeline"
	"github.com/spf13/cobra"
)

func migrateCmd(ctx *app.AppContext) *cobra.Command {
	var dryRun, execute bool
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Import exported data into GitLab",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRun && execute {
				return fmt.Errorf("--dry-run and --execute are mutually exclusive")
			}
			cfg := *ctx.Config
			cfg.DryRun = !execute
			_, err := pipeline.Migrate(&cfg, nil)
			return err
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without writing to GitLab (default)")
	cmd.Flags().BoolVar(&execute, "execute", false, "write to GitLab")
	return cmd
}
