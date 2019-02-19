package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"text/template"

	"github.com/urfave/cli"

	// Required since otherwise dep will prune away these unused packages before codegen has a chance to run
	_ "github.com/99designs/gqlgen/handler"
)

// Execute Run estack
func Execute() {
	loadTemplates()

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

func loadTemplates() {
	filterTemplate = loadTemplateFromFile("models/filter.gotmpl")
	postgresTemplate = loadTemplateFromFile("loader/gen.gotmpl")
	resolverTemplate = loadTemplateFromFile("resolvers/gen.gotmpl")
}

// loadTemplateFromFile Loads template from the package's local directory, under static folder
func loadTemplateFromFile(input string) *template.Template {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}

	source, err := ioutil.ReadFile(path.Dir(filename) + "/static/" + input)
	if err != nil {
		panic(err)
	}

	return template.Must(template.New("").Funcs(templateFuncs).Parse(string(source)))
}
