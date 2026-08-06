package cmd

import (
	"backup-service/internal/app"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run one backup cycle",
	Long:  "Runs a single backup cycle and exits.",
	RunE: func(cmd *cobra.Command, args []string) error {
		appInstance, err := app.New(configPath)
		if err != nil {
			return err
		}
		defer appInstance.Close()

		return appInstance.RunOnce()
	},
}

func init() {
	restoreCmd.Flags().StringVar(&configPath, "config", "", "Config path")
	_ = restoreCmd.MarkFlagRequired("config")

	rootCmd.AddCommand(runCmd)
}
