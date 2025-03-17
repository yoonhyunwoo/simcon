package main

import "github.com/urfave/cli/v2"

var stateCommand = &cli.Command{
	Name:      "state",
	Usage:     "<container-id>",
	ArgsUsage: "<container-id>",
	Action: func(c *cli.Context) error {
		return nil
	},
}
