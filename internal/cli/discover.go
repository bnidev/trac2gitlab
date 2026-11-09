package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/bnidev/trac2gitlab/internal/discover"
	"github.com/spf13/cobra"
)

func discoverCmd(ctx *app.AppContext) *cobra.Command {
	var writeUsers string
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Summarise an exported project and propose authors",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := discover.Run(ctx.Config)
			if err != nil {
				return err
			}
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))

			if writeUsers != "" {
				path := writeUsers
				if !filepath.IsAbs(path) {
					path = filepath.Join(ctx.Config.ProjectDir, path)
				}
				if err := discover.WriteUsers(path, result); err != nil {
					return err
				}
				fmt.Printf("wrote %s\n", path)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&writeUsers, "write-users", "", "write a users.yaml skeleton (relative to the project dir)")
	return cmd
}
