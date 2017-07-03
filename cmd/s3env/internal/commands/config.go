package commands

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "display all config vars",
	Long:  "config fetches all the config vars and displays them in the desired format",
	Run: func(cmd *cobra.Command, args []string) {
		displayVars(s3vars)
	},
}
