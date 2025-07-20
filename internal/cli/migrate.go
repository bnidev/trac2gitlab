package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Import exported data into GitLab",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running migration to GitLab...")
		// TODO: implement GitLab import logic
	},
}

