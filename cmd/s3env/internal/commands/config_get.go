package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var configGetCmd = &cobra.Command{
	Use:   "config:get KEY",
	Short: "display a config value",
	Long:  "config:get fetches the value of a single KEY.",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			displayUsage(cmd)
		}
		fmt.Println(s3vars[args[0]])
	},
}
