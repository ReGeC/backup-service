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
		appInstance, err := app.New(configPath)
		if err != nil {
			return err
		}
		defer appInstance.Close()

		return appInstance.Cleanup()
	},
}

func init() {
	restoreCmd.Flags().StringVar(&configPath, "config", "", "Config path")
	_ = restoreCmd.MarkFlagRequired("config")

	rootCmd.AddCommand(cleanupCmd)
}
