package cli

import (
	"fmt"
	"strings"

	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/bnidev/trac2gitlab/internal/identity"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
	"github.com/spf13/cobra"
)

func usersCmd(ctx *app.AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Inspect the Trac/HG to GitLab user mapping",
	}

	var usersPath string
	var checkGitLab bool
	validate := &cobra.Command{
		Use:   "validate",
		Short: "Validate mappings/users.yaml and optionally resolve GitLab user IDs",
		RunE: func(cmd *cobra.Command, args []string) error {
			mapping, err := identity.LoadFile(usersPath)
			if err != nil {
				return err
			}

			var resolver *identity.Resolver
			if checkGitLab {
				client, err := gitlab.NewGitLabClient(ctx.Config)
				if err != nil {
					return err
				}
				if ctx.Config.GitLab.ProjectID <= 0 {
					return fmt.Errorf("gitlab.project_id is required with --check-gitlab")
				}
				resolver = identity.NewResolver(client, ctx.Config.GitLab.ProjectID)
			}

			users := mapping.Users()
			fmt.Printf("%d users\n\n", len(users))
			for _, user := range users {
				aliases := strings.Join(user.Aliases(), " | ")
				if !user.Resolved() {
					fmt.Printf("✗ %-24s unresolved (aliases: %s)\n", user.Key, aliases)
					continue
				}
				if resolver == nil {
					fmt.Printf("✓ %-24s -> %-20s (aliases: %s)\n", user.Key, user.GitLab, aliases)
					continue
				}
				id, err := resolver.Resolve(user.GitLab)
				if err != nil {
					fmt.Printf("✗ %-24s -> %-20s (%v)\n", user.Key, user.GitLab, err)
					continue
				}
				fmt.Printf("✓ %-24s -> %-20s id=%d\n", user.Key, user.GitLab, id)
			}

			unresolved := mapping.Unresolved()
			if len(unresolved) > 0 {
				return fmt.Errorf("%d unresolved user(s)", len(unresolved))
			}
			return nil
		},
	}
	validate.Flags().StringVar(&usersPath, "users", "mappings/users.yaml", "path to users.yaml")
	validate.Flags().BoolVar(&checkGitLab, "check-gitlab", false, "resolve GitLab usernames to numeric IDs")

	cmd.AddCommand(validate)
	return cmd
}
