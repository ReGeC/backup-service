package cmd

import (
	"backup-service/internal/app"

	"github.com/spf13/cobra"
)

var (
	restoreBackup string
	restoreType string
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore a backup",
	Long:  "Restores a backup from the specified backup file.",
	Example: `  backup-service restore \
    --backup backup_2026-08-06.tar.gz \
    --type postgres`,
	RunE: func(cmd *cobra.Command, args []string) error {
		appInstance, err := app.New(configPath)
		if err != nil {
			return err
		}
		defer appInstance.Close()

		return appInstance.Restore(restoreBackup, restoreType)
	},
}

func init() {
	restoreCmd.Flags().StringVar(&restoreBackup, "backup", "", "Backup file")
	restoreCmd.Flags().StringVar(&restoreType, "type", "", "Backup type")

	_ = restoreCmd.MarkFlagRequired("backup")
	_ = restoreCmd.MarkFlagRequired("type")

	rootCmd.AddCommand(restoreCmd)
}
