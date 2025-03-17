package main

import "github.com/urfave/cli/v2"

var killCommand = &cli.Command{
	Name:      "kill",
	Usage:     "<container-id> <signal>",
	ArgsUsage: "<container-id> <signal>",
	Action: func(c *cli.Context) error {
		return nil
	},
}
