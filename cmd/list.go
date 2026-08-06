package cmd

import (
	"github.com/spf13/cobra"

	"backup-service/internal/app"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all backups",
	Long:  "Displays all available backups.",
	RunE: func(cmd *cobra.Command, args []string) error {
		appInstance, err := app.New()
		if err != nil {
			return err
		}
		defer appInstance.Close()

		return appInstance.List()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
