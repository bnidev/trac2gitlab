package cli

import (
	"fmt"
	"strings"

	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/bnidev/trac2gitlab/internal/importer"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
	"github.com/spf13/cobra"
)

func patchCmd(ctx *app.AppContext) *cobra.Command {
	var container string
	var omnibus, apply bool
	cmd := &cobra.Command{
		Use:   "patch",
		Short: "Emit a Rails-runner script that restores issue and note authors",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ctx.EnsureProjectID(); err != nil {
				return err
			}
			client, err := gitlab.NewGitLabClient(ctx.Config)
			if err != nil {
				return err
			}
			rows, unresolved, err := importer.BuildAuthorPatch(ctx.Config, client)
			if err != nil {
				return err
			}

			ruby := importer.RenderAuthorPatch(rows, ctx.Config.GitLab.ProjectID, apply)
			fmt.Print(importer.RenderShell(ruby, container, omnibus))

			if len(unresolved) > 0 {
				return fmt.Errorf("%d unresolved author(s): %s", len(unresolved), strings.Join(unresolved, ", "))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&container, "container", "gitlab", "GitLab Docker container name")
	cmd.Flags().BoolVar(&omnibus, "omnibus", false, "emit commands for a host install (no docker exec)")
	cmd.Flags().BoolVar(&apply, "apply", false, "set DRY_RUN = false in the emitted script")
	return cmd
}
