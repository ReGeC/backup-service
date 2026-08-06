package cmd

import (
	"backup-service/internal/app"

	"github.com/spf13/cobra"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Cleanup old backups",
	Long:  "Removes old backups according to the configured retention policy.",
	RunE: func(cmd *cobra.Command, args []string) error {
		appInstance, err := app.New()
		if err != nil {
			return err
		}
		defer appInstance.Close()

		return appInstance.Cleanup()
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
}
