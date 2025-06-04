package commands

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/yoonhyunwoo/simcon/pkg/container"
)

// InitCommand Inits a container
func InitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Init a container",
		Action: func(c *cli.Context) error {
			// Get bundle path from environment
			bundle := os.Getenv("_SIMCON_BUNDLE")
			if bundle == "" {
				return fmt.Errorf("bundle path not set")
			}

			// Create container instance
			container, err := container.NewContainer("", bundle)
			if err != nil {
				return fmt.Errorf("failed to create container: %v", err)
			}

			// Setup mounts
			if err := container.InitProcess.SetupMounts(); err != nil {
				return fmt.Errorf("failed to setup mounts: %v", err)
			}

			// Setup security configurations
			if err := container.InitProcess.SetupSecurity(); err != nil {
				return fmt.Errorf("failed to setup security: %v", err)
			}

			// Run the container process
			return container.InitProcess.CreateProcess()
		},
	}
}
