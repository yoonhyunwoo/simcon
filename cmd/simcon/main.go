package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/yoonhyunwoo/simcon/cmd/simcon/commands"
)

func main() {
	app := &cli.App{
		Name:  "simcon",
		Usage: "A simple OCI container runtime",
		Commands: []*cli.Command{
			commands.CreateCommand(),
			commands.StartCommand(),
			commands.KillCommand(),
			commands.DeleteCommand(),
			commands.StateCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
