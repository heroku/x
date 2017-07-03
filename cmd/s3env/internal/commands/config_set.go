package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var configSetCmd = &cobra.Command{
	Use:   "config:set KEY1=VAL1 KEY2=VAL2",
	Short: "set one or more config vars",
	Long:  "config:set sets one or more config vars",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			displayUsage(cmd)
		}

		vars, err := parseEnvironStrings(args)
		if err != nil {
			displayErr(err)
		}

		var keys []string
		for k, v := range vars {
			keys = append(keys, k)
			s3vars[k] = v
		}

		fmt.Printf("Setting %s... ", strings.Join(keys, ", "))
		if err := persistVars(); err != nil {
			displayErr(err)
		}
		fmt.Println("done!")
		displayVars(vars)
	},
}
