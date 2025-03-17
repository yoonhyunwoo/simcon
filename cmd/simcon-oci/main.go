package main

import (
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

const (
	configPath = "config.json"
)

func main() {
	app := &cli.App{
		Name:  "simcon-oci",
		Usage: "oci implementation of simcon project",
		Action: func(c *cli.Context) error {
			return nil
		},
	}

	app.Commands = []*cli.Command{
		stateCommand,
		createCommand,
		startCommand,
		killCommand,
		deleteCommand,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
