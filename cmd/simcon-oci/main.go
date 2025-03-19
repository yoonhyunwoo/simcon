package main

import (
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

const (
	configFile = "config.json"
	dataDir    = "/var/lib/simcon-oci"
	ociVersion = "1.0.0"
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
