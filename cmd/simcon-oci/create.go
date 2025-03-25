package main

import (
	"github.com/urfave/cli/v2"
	"log"
)

var createCommand = &cli.Command{
	Name:      "create",
	Usage:     "<container-id> <path-to-bundle>",
	ArgsUsage: "<container-id> <path-to-bundle>",
	Action: func(context *cli.Context) error {

		if err := validateArgs(context, 2, exactArgs); err != nil {
			return err
		}

		status, err := create(context)
		if err != nil {
			return err
		}

		if status == 1 {
			log.Println(context.Args().First())
		}

		log.Println(context.Args().First())

		return nil
	},
}
