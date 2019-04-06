package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/codemodus/kace"
	"github.com/urfave/cli"
	gcli "gnorm.org/gnorm/cli"
	"gnorm.org/gnorm/environ"
)

var resolverTemplate *template.Template
var postgresTemplate *template.Template
var filterTemplate *template.Template

var genCmd = cli.Command{
	Name:  "generate",
	Usage: "generate estack files",
	Flags: []cli.Flag{
		cli.StringFlag{Name: "config, c", Usage: "the config filename"},
		cli.StringFlag{Name: "folder", Usage: "where to create the project"},
	},
	Action: func(ctx *cli.Context) {
		if len(ctx.String("folder")) > 0 {
			err := os.Chdir(ctx.String("folder"))
			if err != nil {
				exit(err)
			}
		}

		// Ensure package name is loaded:
		err := loadPackageName()
		if err != nil {
			exit(err)
		}

		// Load config.yaml
		config, err := readConfig(filePath(ctx, "config.yaml"))
		if err != nil {
			exit(err)
		}

		log.Printf("Config:\n%+v", config)

		generateGnorm()

		var tasks []Task
		tasks = append(tasks, Task{Folder: "loader", Build: postgresBuild})
		tasks = append(tasks, Task{Folder: "models", Build: modelsBuild})
		tasks = append(tasks, Task{Folder: "resolvers", Build: resolverBuild})
		generateFiles(ctx, config, tasks)

		// Recreate GraphQL Code
		_ = generateGQL(ctx)
	},
}

func generateGnorm() {
	env := environ.Values{
		Args:   []string{"gen"},
		Stderr: os.Stderr,
		Stdout: os.Stdout,
		Stdin:  os.Stdin,
	}

	// Delete any existing gnorm files so there are no legacy ones around
	runCommand("rm -rf gnorm")
	copyTemplate("templates/table.gotmpl", "templates/table.gotmpl")
	copyTemplate("templates/enum.gotmpl", "templates/enum.gotmpl")
	copyTemplate("templates/schema.gotmpl", "templates/schema.gotmpl")

	gcli.ParseAndRun(env)

	runCommand(fmt.Sprintf("mv %s %s", "gnorm/Public/tables", "gnorm/dbl"))
	runCommand(fmt.Sprintf("rm -rf %s", "gnorm/Public"))
	runCommand(fmt.Sprintf("goimports -w %s", "gnorm/."))
	runCommand(fmt.Sprintf("goimports -w %s", "gnorm/dbl/."))

	copyTemplate("gnorm/db.go", "gnorm/db.go")
	copyTemplate("gnorm/where.go", "gnorm/where.go")
	copyTemplate("gnorm/dbl/util.go", "gnorm/dbl/util.go")

}

// copyTemplate Copies files from the template folder relative to *this* file
// to the destination
func copyTemplate(source string, destination string) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}

	input, err := ioutil.ReadFile(path.Dir(filename) + "/static/" + source)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(destination, input, 0644)
	if err != nil {
		panic(err)
	}
}

var authorisationModels = []string{
	"Address",
	"Brand",
	"Client",
	"Invoice",
	"Person",
	"Transaction",
}

// Link Used for auto-generating links between particular items
type Link struct {
	Model1 string
	Model2 string
}

var links = []Link{}

var templateFuncs = map[string]interface{}{
	"camel":        kace.Camel,
	"concat":       concat,
	"compare":      strings.Compare,
	"contains":     strings.Contains,
	"containsAny":  strings.ContainsAny,
	"count":        strings.Count,
	"equalFold":    strings.EqualFold,
	"fields":       strings.Fields,
	"hasPrefix":    strings.HasPrefix,
	"hasSuffix":    strings.HasSuffix,
	"strIndex":     strings.Index,
	"indexAny":     strings.IndexAny,
	"join":         strings.Join,
	"kebab":        kace.Kebab,
	"kebabUpper":   kace.KebabUpper,
	"lastIndex":    strings.LastIndex,
	"lastIndexAny": strings.LastIndexAny,
	"pascal":       kace.Pascal,
	"repeat":       strings.Repeat,
	"replace":      strings.Replace,
	"snake":        kace.Snake,
	"snakeUpper":   kace.SnakeUpper,
	"split":        strings.Split,
	"splitAfter":   strings.SplitAfter,
	"splitAfterN":  strings.SplitAfterN,
	"splitN":       strings.SplitN,
	"title":        strings.Title,
	"toLower":      strings.ToLower,
	"toTitle":      strings.ToTitle,
	"toUpper":      strings.ToUpper,
	"trim":         strings.Trim,
	"trimLeft":     strings.TrimLeft,
	"trimPrefix":   strings.TrimPrefix,
	"trimRight":    strings.TrimRight,
	"trimSpace":    strings.TrimSpace,
	"trimSuffix":   strings.TrimSuffix,
}

func concat(vals ...string) string {
	return strings.Join(vals, "")
}

// Task We go through a few folders, deleting generated files and running the template
type Task struct {
	Folder string
	Build  func(config Config, folder string) error
}

func generateFiles(ctx *cli.Context, config Config, tasks []Task) {
	// Set up the tasks:

	for _, t := range tasks {
		path := filePath(ctx, t.Folder)
		// Delete ALL previously generated files
		cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("rm %s/gen_*.go", path))
		err := cmd.Run()
		if err != nil {
			log.Printf("Failed to delete existing files in %s with '%s', but continuing...", path, err)
		}

		die(t.Build(config, path))
	}
}

func die(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func authorisationBuild(folder string) error {
	fileName := "gen_models.go"
	if len(folder) > 0 {
		fileName = folder + "/" + fileName
	}
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}

	err = authorisationTemplate.Execute(f, struct {
		Timestamp time.Time
		Models    []string
	}{
		Timestamp: time.Now(),
		Models:    authorisationModels,
	})
	f.Close()

	if err != nil {
		return err
	}

	return goImports(fileName)
}

func modelsBuild(config Config, folder string) error {
	fileName := "filter.go"
	if len(folder) > 0 {
		fileName = folder + "/" + fileName
	}
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}

	err = filterTemplate.Execute(f, struct {
		Timestamp time.Time
		Config    Config
	}{
		Timestamp: time.Now(),
		Config:    config,
	})
	f.Close()

	if err != nil {
		return err
	}

	return goImports(fileName)
}

func postgresBuild(config Config, folder string) error {
	// Core models
	for _, b := range config.Generate.Postgres {
		fileName := fmt.Sprintf("gen_%s.go", kace.Snake(b.ModelName))

		if len(folder) > 0 {
			fileName = folder + "/" + fileName
		}

		f, err := os.Create(fileName)

		if err != nil {
			return err
		}

		err = postgresTemplate.Execute(f, struct {
			Config            Config
			Timestamp         time.Time
			ModelName         string
			ModelPackage      string
			ModelPackageShort string
			PmName            string
			PK                string
			PrimaryKeyType    string
			Create            bool
		}{
			Config:            config,
			Timestamp:         time.Now(),
			ModelName:         b.ModelName,
			ModelPackage:      b.ModelPackage,
			ModelPackageShort: b.ModelPackageShort,
			PmName:            b.PmName,
			PK:                b.PK,
			PrimaryKeyType:    b.PrimaryKeyType,
			Create:            b.Create,
		})
		f.Close()

		if err != nil {
			return err
		}

		err = goImports(fileName)
		if err != nil {
			return err
		}
	}

	// Links:
	fileName := fmt.Sprintf("gen_links.go")

	if len(folder) > 0 {
		fileName = folder + "/" + fileName
	}

	f, err := os.Create(fileName)

	if err != nil {
		return err
	}

	err = linkTemplate.Execute(f, struct {
		Config    Config
		Timestamp time.Time
		Models    []Link
	}{
		Config:    config,
		Timestamp: time.Now(),
		Models:    links,
	})
	f.Close()

	if err != nil {
		return err
	}

	err = goImports(fileName)
	if err != nil {
		return err
	}

	return nil
}

func resolverBuild(config Config, folder string) error {
	log.Printf("Creating resolver files")
	for _, b := range config.Generate.Resolvers {
		log.Printf("...%s", b.SingularModelName)
		fileName := fmt.Sprintf("gen_%s.go", kace.Snake(b.SingularModelName))

		if len(folder) > 0 {
			fileName = folder + "/" + fileName
		}

		f, err := os.Create(fileName)

		if err != nil {
			return err
		}

		err = resolverTemplate.Execute(f, struct {
			Config          Config
			Timestamp       time.Time
			ModelName       string
			PluralModelName string
			Create          bool
			Update          bool
			PrepareCreate   bool
			Query           bool
		}{
			Config:          config,
			Timestamp:       time.Now(),
			ModelName:       b.SingularModelName,
			PluralModelName: b.PluralModelName,
			Create:          b.Create,
			Update:          b.Update,
			PrepareCreate:   b.PrepareCreate,
			Query:           b.Query,
		})
		f.Close()

		if err != nil {
			return err
		}

		err = goImports(fileName)
		if err != nil {
			return err
		}
	}

	return nil
}

func goImports(fileName string) error {
	// Run goimports against newly created file:
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("goimports -w %s", fileName))
	return cmd.Run()
}

var authorisationTemplate = template.Must(template.New("").Funcs(templateFuncs).Parse(
	`// Code generated by go generate; DO NOT EDIT.
// This file was generated by robots
package authorisation

import (
	"{{.Config.PackageName}}/loader"
	"{{.Config.PackageName}}/models"
	"{{.Config.PackageName}}/gnorm/dbl"
	"github.com/episub/estack/opa"
	opentracing "github.com/opentracing/opentracing-go"
)

{{ range $x, $c :=  .Models -}}
// {{.}}Input Create {{.}}Input as a variable so that it can be overridden in the init function if desired
var {{.}}Input = func(ctx context.Context, input map[string]interface{}, i models.{{.}}) error {
	input["{{camel .}}"] = i
	input["user"] = GetUserFromContext(ctx)

	return nil
}

// {{.}}Fetch Fetches {{.}} and authorises
func {{.}}Fetch(ctx context.Context, id string) (*models.{{.}}, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "{{.}}Fetch")
	defer span.Finish()
	o, err := loader.Get{{.}}(ctx, id)
	if err != nil {
		return nil, err
	}
	return {{.}}(ctx, o)
}

// {{.}} Authorises {{.}}
func {{.}}(ctx context.Context, i models.{{.}}) (*models.{{.}}, error) {
	input := make(map[string]interface{})
	err := AddDefaultPayload(ctx, input)
	if err != nil {
		return nil, err
	}
	{{.}}Input(ctx, input, i)

	allowed, err := opa.Authorised(ctx, getAuthString("query", "{{camel .}}", "allow"), input)

	if err != nil {
		return nil, err
	}

	if !allowed {
		return nil, permissionDeniedError("{{camel .}}")
	}

	return &i, nil
}
{{ end }}
`))

var linkTemplate = template.Must(template.New("").Funcs(templateFuncs).Parse(
	`// Code generated by go generate; DO NOT EDIT.
// This file was generated by robots
package loader

import (
	"{{.Config.PackageName}}/models"
	"{{.Config.PackageName}}/loader"
	"{{.Config.PackageName}}/gnorm"
	"{{.Config.PackageName}}/gnorm/dbl"
	"github.com/episub/estack/validate"
	"github.com/episub/estack/opa"
	opentracing "github.com/opentracing/opentracing-go"
)

{{ range $x, $c :=  .Models -}}
{{$m1 := $c.Model1}}
{{$m2 := $c.Model2}}
{{$full := concat $m1 $m2}}
// Link{{$c.Model1}}{{$c.Model2}} Links '{{$c.Model1}}' to {{$c.Model2}}'
func (l *PostgresLoader) Link{{$c.Model1}}{{$c.Model2}}(ctx context.Context, {{camel $m1}}ID string, {{camel $m2}}ID string, link bool) (bool, error) {
	if link {
		var clink dbl.{{$m1}}{{$m2}}

		clink.{{$c.Model1}}ID{{$c.Model1}} = {{camel $c.Model1}}ID
		clink.{{$c.Model2}}ID{{$c.Model2}} = {{camel $c.Model2}}ID

		_, err := dbl.Upsert{{$m1}}{{$m2}}(ctx, l.pool, clink)

		// Sanitise our output, and log errors if needed:
		err = sanitiseError(err)

		return (err == nil), err
	}

	// !link, therefore delete any such connection:
	res, err := l.pool.Exec("DELETE FROM dbl.{{snake $full}} WHERE {{snake $c.Model1}}_id_{{snake $c.Model1}}=$1 AND {{snake $c.Model2}}_id_{{snake $c.Model2}}=$2", {{camel $m1}}ID, {{camel $m2}}ID)

	err = sanitiseError(err)
	if err == nil && res.RowsAffected() == 0 {
		err = fmt.Errorf("No such link exists")
	}
	return (err == nil), err
}
{{end}}
`))

//var resolverTemplate = template.Must(template.New("").Funcs(templateFuncs).Parse(
//	`// Code generated by go generate; DO NOT EDIT.
//// This file was generated by robots
//package resolvers
//
//import (
//	"{{.Config.PackageName}}/models"
//	"{{.Config.PackageName}}/loader"
//	"{{.Config.PackageName}}/gnorm"
//	"github.com/episub/estack/opa"
//	"github.com/99designs/gqlgen/graphql"
//	"github.com/vektah/gqlparser/gqlerror"
//	opentracing "github.com/opentracing/opentracing-go"
//)
//
//func (r *queryResolver) EditableUpdate{{.ModelName}}Fields(ctx context.Context, id string) ([]string, error) {
//	return editableUpdate{{.ModelName}}Fields(ctx, id)
//}
//
//{{if .Update}}
//// Update{{.ModelName}} Updates {{.ModelName}} with provided input
//func (r *mutationResolver) Update{{.ModelName}}(ctx context.Context, id string, u map[string]interface{}) (*models.{{.ModelName}}, error) {
//	// Get allowed edit fields:
//	allowed, err := editableUpdate{{.ModelName}}Fields(ctx, id)
//	if err != nil {
//		return nil, err
//	}
//
//	// Filter out unapproved changes
//	changes, _, any, err := authorisedChanges(ctx, allowed, u)
//
//	if err != nil {
//		return nil, err
//	}
//
//	if !any {
//		return nil, fmt.Errorf("No fields were permitted to be updated")
//	}
//
//	err = loader.Update{{.ModelName}}(ctx, id, changes)
//
//	if err != nil {
//		return nil, err
//	}
//
//	obj, err := loader.Get{{.ModelName}}(ctx, id)
//
//	return &obj, err
//}
//{{end}}
//{{if .PrepareCreate}}
//// prepareCreate{{.ModelName}} Performs some pre-processing on the provided map.  Disable generation of this function and create your own if you require any pre-processing performed
//func prepareCreate{{.ModelName}}(ctx context.Context, i map[string]interface{}) error {
//	return nil
//}
//{{end}}
//{{if .Create}}
//// Create{{.ModelName}} Creates {{.ModelName}}
//func (r *mutationResolver) Create{{.ModelName}}(ctx context.Context, i map[string]interface{}) (*models.{{.ModelName}}, error) {
//	err := prepareCreate{{.ModelName}}(ctx, i)
//	if err != nil {
//		return nil, err
//	}
//
//	id, err := loader.Create{{.ModelName}}(ctx, i)
//
//	if err != nil {
//		return nil, err
//	}
//
//	obj, err := loader.Get{{.ModelName}}(ctx, id)
//	return &obj, err
//}
//{{end}}
//{{if .Query}}
//func query{{.PluralModelName}}(ctx context.Context, first *int, after *string, last *int, before *string, cf *models.{{.ModelName}}Filter, sortField *models.{{.ModelName}}Sort, sortDirection *models.SortDirection, where []gnorm.WhereClause) (o models.{{.PluralModelName}}Connection, err error) {
//	span, ctx := opentracing.StartSpanFromContext(ctx, "query{{.PluralModelName}}")
//	defer span.Finish()
//
//	f := models.NewFilter(first, after, last, before, sortDirection)
//
//	// Set up the sort order based on inputs:
//	if sortField != nil {
//		var err error
//
//		f.Order, err = sort{{.ModelName}}(ctx, *sortField, f.Order)
//
//		if err != nil {
//			return o, fmt.Errorf("Cannot sort by field %s: %s", sortField, err)
//		}
//	}
//
//	// Configure the where clauses:
//	if cf != nil {
//		var fw []gnorm.WhereClause
//		fw, err = filter{{.ModelName}}(ctx, *cf)
//		if err != nil {
//			return
//		}
//
//		where = append(where, fw...)
//	}
//
//	f.Where = where
//
//	r, pi, count, err := loader.GetAll{{.ModelName}}(ctx, f)
//
//	if err != nil {
//		return o, err
//	}
//
//	o.PageInfo = pi
//	o.TotalCount = count
//	o.Edges = make([]models.{{.ModelName}}Edge, len(r))
//
//	for i, t := range r {
//		o.Edges[i] = models.{{.ModelName}}Edge{Cursor: t.ID, Node: t}
//	}
//
//	return o, err
//}
//{{end}}
//`))
