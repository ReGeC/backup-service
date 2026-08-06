package cmd

import (
	"backup-service/internal/app"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start backup scheduler",
	Long:  "Starts the backup scheduler and runs it continuously.",
	RunE: func(cmd *cobra.Command, args []string) error {
		appInstance, err := app.New()
		if err != nil {
			return err
		}
		defer appInstance.Close()

		return appInstance.Start()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
