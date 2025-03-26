package main

import "github.com/urfave/cli/v2"

// spec command is setup in container
var specCommand = &cli.Command{
	Name: "spec",
	Action: func(context *cli.Context) error {
		defaultSpec()
		return nil
	},
}
