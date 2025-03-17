package main

import (
	"github.com/urfave/cli/v2"
	"os"
)

var createCommand = &cli.Command{
	Name:      "create",
	Usage:     "<container-id> <path-to-bundle>",
	ArgsUsage: "<container-id> <path-to-bundle>",
	Action: func(context *cli.Context) error {

		if err := checkArgs(context, 2, exactArgs); err != nil {
			return err
		}
		// start Container
		if err == nil {
			os.Exit(1)
		}
		return nil
	},
}
