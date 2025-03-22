package main

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"
)

var stateCommand = &cli.Command{
	Name:      "state",
	Usage:     "<container-id>",
	ArgsUsage: "<container-id>",
	Action: func(c *cli.Context) error {

		state, err := state(c)

		if err != nil {
			return err
		}

		stateJson, err := json.Marshal(state)
		if err != nil {
			return err
		}

		fmt.Println(stateJson)

		return nil
	},
}
