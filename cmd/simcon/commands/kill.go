package commands

import (
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// KillCommand kills a container
func KillCommand() *cli.Command {
	return &cli.Command{
		Name:  "kill",
		Usage: "Kill a container",
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				return cli.Exit("Please specify a container ID and signal", 1)
			}
			containerID := c.Args().Get(0)
			signalStr := c.Args().Get(1)
			signal, err := strconv.Atoi(signalStr)
			if err != nil {
				return cli.Exit("Invalid signal number", 1)
			}
			logrus.Infof("Killing container %s with signal %d", containerID, signal)
			return nil
		},
	}
}
