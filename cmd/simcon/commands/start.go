package commands

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/yoonhyunwoo/simcon/pkg/container"
)

// StartCommand starts a container
func StartCommand() *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "Start a container",
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return cli.Exit("Please specify a container ID", 1)
			}
			containerID := c.Args().Get(0)
			stateManager := container.NewStateManager()
			state, err := stateManager.GetState(containerID)
			if err != nil {
				return cli.Exit(fmt.Sprintf("Failed to get container state: %v", err), 1)
			}

			container, err := container.NewContainer(containerID, state.Bundle)
			if err != nil {
				return cli.Exit(fmt.Sprintf("Failed to create container: %v", err), 1)
			}

			err = container.Start()
			if err != nil {
				return cli.Exit(fmt.Sprintf("Failed to start container: %v", err), 1)
			}

			logrus.Infof("Starting container %s", containerID)
			return nil
		},
	}
}
