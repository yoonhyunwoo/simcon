package main

import "github.com/urfave/cli/v2"

var deleteCommand = &cli.Command{
	Name:      "delete",
	Usage:     "<container-id>",
	ArgsUsage: "<container-id>",
	Action: func(c *cli.Context) error {
		return nil
	},
}
