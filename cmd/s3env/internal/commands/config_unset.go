package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var configUnsetCmd = &cobra.Command{
	Use:   "config:unset KEY1 [KEY2]...",
	Short: "unset one or more config vars",
	Long:  "config:unset unsets one or more config vars",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			displayUsage(cmd)
		}

		for _, key := range args {
			delete(s3vars, key)
		}

		fmt.Printf("Unsetting %s... ", strings.Join(args, ", "))
		if err := persistVars(); err != nil {
			displayErr(err)
		}
		fmt.Println("done!")
	},
}
