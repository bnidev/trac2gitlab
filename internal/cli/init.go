package cli

import (
	"trac2gitlab/internal/config"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the Trac to GitLab migration environment by creating a default configuration",
	Run: func(cmd *cobra.Command, args []string) {
		configExists := config.CheckConfigExists()
		if configExists {
			cmd.Println("Configuration already exists. Use 'trac2gitlab export' to start exporting data.")
			return
		}
		if err := config.CreateDefaultConfig(); err != nil {
			cmd.Println("❌ Failed to create default configuration:", err)
			return
		}
		cmd.Println("Default configuration created successfully.")
	},
}
