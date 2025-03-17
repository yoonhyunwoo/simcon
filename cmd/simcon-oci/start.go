package main

import "github.com/urfave/cli/v2"

var startCommand = &cli.Command{
	Name:      "start",
	Usage:     "<container-id>",
	ArgsUsage: "<container-id>",
	Action: func(c *cli.Context) error {
		return nil
	},
}
