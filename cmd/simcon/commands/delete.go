package commands

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// DeleteCommand deletes a container
func DeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a container",
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return cli.Exit("Please specify a container ID", 1)
			}
			containerID := c.Args().Get(0)
			logrus.Infof("Deleting container %s", containerID)
			return nil
		},
	}
}
