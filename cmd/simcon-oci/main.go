package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

const (
	configFile     = "config.json"
	dataDir        = "/var/lib/simcon-oci"
	ociVersion     = "1.0.0"
	stateFile      = "state.json"
	cgroupBasePath = "/sys/fs/cgroup"
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
		specCommand,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
