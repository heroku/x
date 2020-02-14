// Package hypercmd provides utilities for creating "hyper commands", where
// multiple commands are bundled into a single executable to get faster builds
// and smaller binaries.
package hypercmd

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"

	cli "github.com/urfave/cli"
)

// New configures a cli.Command for executing a hyper command service.
func New(name string, fn func()) cli.Command {
	return cli.Command{
		Name:   name,
		Usage:  "Run " + name,
		Before: serviceEnvLoader(name),
		Action: func(c *cli.Context) error {
			fn()
			return nil
		},
	}
}

// serviceEnvLoader will load JSON-format environment files before executing
// the service function.
func serviceEnvLoader(name string) cli.BeforeFunc {
	return func(c *cli.Context) error {
		data, err := ioutil.ReadFile(".env." + name)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}

			return errors.Wrap(err, "load env")
		}

		var env map[string]string
		if err := json.Unmarshal(data, &env); err != nil {
			return errors.Wrap(err, "parse env")
		}

		for k, v := range env {
			if err := os.Setenv(k, v); err != nil {
				return errors.Wrap(err, "set env")
			}
		}

		return nil
	}
}
