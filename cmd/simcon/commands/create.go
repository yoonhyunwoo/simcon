package commands

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/yoonhyunwoo/simcon/pkg/container"
)

// CreateCommand creates a new container
func CreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a container",
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				return cli.Exit("Please specify a container ID and bundle path", 1)
			}
			containerID := c.Args().Get(0)
			bundle := c.Args().Get(1)
			logrus.Infof("Creating container %s from bundle %s", containerID, bundle)
			container, err := container.NewContainer(containerID, bundle)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			err = container.Create()
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			logrus.Infof("Container created: %+v", container)
			return nil
		},
	}
}
