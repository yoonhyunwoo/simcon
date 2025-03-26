package main

import "github.com/urfave/cli/v2"

// init command is setup in container
var initCommand = cli.Command{
	Name: "init",
	Action: func(context *cli.Context) error {
		if err := containerInit(context); err != nil {
			return err
		}
		return nil
	},
}
