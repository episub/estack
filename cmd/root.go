package cmd

import (
	"fmt"
	"os"

	"github.com/urfave/cli"

	// Required since otherwise dep will prune away these unused packages before codegen has a chance to run
	_ "github.com/99designs/gqlgen/handler"
)

// Execute Run estack
func Execute() {
	app := cli.NewApp()
	app.Name = "estack"
	app.Usage = genCmd.Usage
	app.Description = "Tools and libraries for setting up and maintaining a project using the Episub Stack."
	app.HideVersion = true
	app.Flags = genCmd.Flags
	app.Action = genCmd.Action
	app.Commands = []cli.Command{
		genCmd,
		initCmd,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}
