package commands

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// StateCommand gets container state
func StateCommand() *cli.Command {
	return &cli.Command{
		Name:  "state",
		Usage: "Get container state",
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return cli.Exit("Please specify a container ID", 1)
			}
			containerID := c.Args().Get(0)
			logrus.Infof("Getting state for container %s", containerID)
			return nil
		},
	}
}
