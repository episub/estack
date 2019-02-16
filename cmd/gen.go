package cmd

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/codemodus/kace"
	"github.com/urfave/cli"
)

var genCmd = cli.Command{
	Name:  "generate",
	Usage: "generate a graphql server based on schema",
	Flags: []cli.Flag{
		cli.BoolFlag{Name: "verbose, v", Usage: "show logs"},
		cli.StringFlag{Name: "config, c", Usage: "the config filename"},
	},
	Action: func(ctx *cli.Context) {
		_ = generateGQL()
	},
}

var authorisationModels = []string{
	"Address",
	"Brand",
	"Client",
	"Invoice",
	"Person",
	"Transaction",
}

var postgresModels = []struct {
	ModelName string
	PmName    string
	PK        string
	Create    bool
}{
	{
		ModelName: "Address",
		PmName:    "Address",
		PK:        "AddressID",
		Create:    true,
	},
	{
		ModelName: "Area",
		PmName:    "Area",
		PK:        "AreaID",
	},
	{
		ModelName: "Brand",
		PmName:    "Brand",
		PK:        "BrandID",
	},
	{
		ModelName: "BankAccount",
		PmName:    "BankAccount",
		PK:        "BankAccountID",
		Create:    true,
	},
	{
		ModelName: "CeasedReason",
		PmName:    "CeasedReason",
		PK:        "CeasedReasonID",
	},
	{
		ModelName: "Client",
		PmName:    "Client",
		PK:        "ClientID",
		Create:    true,
	},
	{
		ModelName: "Country",
		PmName:    "Country",
		PK:        "CountryID",
	},
	{
		ModelName: "File",
		PmName:    "File",
		PK:        "FileID",
		Create:    true,
	},
	{
		ModelName: "Gender",
		PmName:    "Gender",
		PK:        "GenderID",
	},
	{
		ModelName: "Invoice",
		PmName:    "Invoice",
		PK:        "InvoiceID",
		Create:    true,
	},
	{
		ModelName: "Note",
		PmName:    "Note",
		PK:        "NoteID",
		Create:    true,
	},
	{
		ModelName: "PaymentMethod",
		PmName:    "PaymentMethod",
		PK:        "PaymentMethodID",
	},
	{
		ModelName: "Person",
		PmName:    "Person",
		PK:        "PersonID",
		Create:    true,
	},
	{
		ModelName: "Provider",
		PmName:    "Provider",
		PK:        "ProviderID",
	},
	{
		ModelName: "ServiceType",
		PmName:    "ServiceType",
		PK:        "ServiceTypeID",
	},
	{
		ModelName: "Session",
		PmName:    "Session",
		PK:        "SessionID",
	},
	{
		ModelName: "State",
		PmName:    "State",
		PK:        "StateID",
	},
	{
		ModelName: "Supplier",
		PmName:    "Supplier",
		PK:        "SupplierID",
		Create:    true,
	},
	{
		ModelName: "Supplement",
		PmName:    "Supplement",
		PK:        "SupplementID",
	},
	{
		ModelName: "Transaction",
		PmName:    "Transaction",
		PK:        "TransactionID",
		Create:    true,
	},
	{
		ModelName: "TransactionType",
		PmName:    "TransactionType",
		PK:        "TransactionTypeID",
		//Create:    true,
	},
	{
		ModelName: "User",
		PmName:    "User",
		PK:        "UserID",
		Create:    true,
	},
}

var resolverModels = []struct {
	SingularModelName string
	PluralModelName   string
	Create            bool // Build a create function
	Update            bool // Build an update function
	PrepareCreate     bool // Provide a prepare function for you (set to false if you want to set one yourself)
	Query             bool // Creates a queryX function used for pagination via a connections type method
}{
	{
		SingularModelName: "Client",
		PluralModelName:   "Clients",
		Create:            true,
		Update:            true,
		PrepareCreate:     true,
		Query:             true,
	},
	{
		SingularModelName: "Invoice",
		PluralModelName:   "Invoices",
		Create:            true,
		Update:            true,
		Query:             true,
	},
	{
		SingularModelName: "Note",
		PluralModelName:   "Notes",
		Update:            true,
		PrepareCreate:     true,
	},
	{
		SingularModelName: "Supplier",
		PluralModelName:   "Suppliers",
		Create:            true,
		Update:            true,
		PrepareCreate:     true,
		Query:             true,
	},
	{
		SingularModelName: "Transaction",
		PluralModelName:   "Transactions",
		Create:            true,
		Update:            true,
		PrepareCreate:     true,
		Query:             true,
	},
}

// Link Used for auto-generating links between particular items
type Link struct {
	Model1 string
	Model2 string
}

var links = []Link{
	{
		Model1: "ServiceType",
		Model2: "Supplier",
	},
	{
		Model1: "Area",
		Model2: "Supplier",
	},
	{
		Model1: "Client",
		Model2: "Supplier",
	},
}

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
	Build  func(folder string) error
}

var tasks []Task

func main() {
	// Set up the tasks:
	//tasks = append(tasks, Task{Folder: "authorisation", Build: authorisationBuild})
	tasks = append(tasks, Task{Folder: "loaders/postgres", Build: postgresBuild})
	tasks = append(tasks, Task{Folder: "resolvers", Build: resolverBuild})

	for _, t := range tasks {
		// Delete ALL previously generated files
		cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("rm %s/gen_*.go", t.Folder))
		err := cmd.Run()
		if err != nil {
			log.Printf("Failed to delete existing files in %s with '%s', but continuing...", t.Folder, err)
		}

		die(t.Build(t.Folder))
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

func postgresBuild(folder string) error {
	// Core models
	for _, b := range postgresModels {
		fileName := fmt.Sprintf("gen_%s.go", kace.Snake(b.ModelName))

		if len(folder) > 0 {
			fileName = folder + "/" + fileName
		}

		f, err := os.Create(fileName)

		if err != nil {
			return err
		}

		err = postgresTemplate.Execute(f, struct {
			Timestamp time.Time
			ModelName string
			PmName    string
			PK        string
			Create    bool
		}{
			Timestamp: time.Now(),
			ModelName: b.ModelName,
			PmName:    b.PmName,
			PK:        b.PK,
			Create:    b.Create,
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
		Timestamp time.Time
		Models    []Link
	}{
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

func resolverBuild(folder string) error {
	for _, b := range resolverModels {
		fileName := fmt.Sprintf("gen_%s.go", kace.Snake(b.SingularModelName))

		if len(folder) > 0 {
			fileName = folder + "/" + fileName
		}

		f, err := os.Create(fileName)

		if err != nil {
			return err
		}

		err = resolverTemplate.Execute(f, struct {
			Timestamp       time.Time
			ModelName       string
			PluralModelName string
			Create          bool
			Update          bool
			PrepareCreate   bool
			Query           bool
		}{
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
	"bitbucket.org/replaceme/api/loaders"
	"bitbucket.org/replaceme/api/models"
	"bitbucket.org/replaceme/api/opa"
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
	o, err := loaders.Current.Get{{.}}(ctx, id)
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

var postgresTemplate = template.Must(template.New("").Funcs(templateFuncs).Parse(
	`// Code generated by go generate; DO NOT EDIT.
// This file was generated by robots
package postgres

import (
	"bitbucket.org/replaceme/api/models"
	"bitbucket.org/replaceme/api/validate"
	"bitbucket.org/replaceme/gnorm/gnorm"
	"bitbucket.org/replaceme/gnorm/gnorm/pm"
	"github.com/codemodus/kace"
	opentracing "github.com/opentracing/opentracing-go"
)

// {{.ModelName}}FetchRequest A request for a {{camel .ModelName}} object, to be batched
type {{.ModelName}}FetchRequest struct {
	{{.ModelName}}ID string
	Reply    chan {{.ModelName}}FetchReply
}

// {{.ModelName}}FetchReply A reply with the requested object or an error
type {{.ModelName}}FetchReply struct {
	{{.ModelName}} pm.{{.ModelName}}Full
	Error  error
}

var {{camel .ModelName}}Initialised bool
var {{camel .ModelName}}FRs []{{.ModelName}}FetchRequest
var {{camel .ModelName}}MX sync.Mutex

// Get{{.ModelName}} Returns models.{{.ModelName}} with given ID
{{- $idName := snake .ModelName}}
func (l *Loader) Get{{.ModelName}}(ctx context.Context, id string) (o models.{{.ModelName}}, err error) {
	return l.get{{.ModelName}}(ctx, id, l.pool)
}

// get{{.ModelName}} Returns models.{{.ModelName}} with given ID, using provided DB connection
func (l *Loader) get{{.ModelName}}(ctx context.Context, id string, db gnorm.DB) (o models.{{.ModelName}}, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Get{{.ModelName}}")
	defer span.Finish()

	r, err := l.batchedGet{{.PmName}}(id, l.pool)

	if err != nil {
		err = sanitiseError(err)
		return
	}

	o = hydrateModel{{.ModelName}}(ctx, r)

	return
}

func (l *Loader) batchedGet{{.ModelName}}(id string, db gnorm.DB) (o pm.{{.PmName}}Full, err error) {
	{{camel .ModelName}}MX.Lock()
	if !{{camel .ModelName}}Initialised {
		err = fmt.Errorf("batchedGet{{.ModelName}} not initialised.  Add 'go loader.run{{.ModelName}}Batcher()' to init")
	}
	{{camel .ModelName}}MX.Unlock()
	if err != nil {
		return
	}

	rchan := make(chan {{.ModelName}}FetchReply)
	r := {{.ModelName}}FetchRequest{
		{{.ModelName}}ID: id,
		Reply:    rchan,
	}

	{{camel .ModelName}}MX.Lock()
	{{camel .ModelName}}FRs = append({{camel .ModelName}}FRs, r)
	{{camel .ModelName}}MX.Unlock()

	reply := <-rchan

	return reply.{{.ModelName}}, reply.Error
}

func (l *Loader) run{{.ModelName}}Batcher() {
	{{camel .ModelName}}MX.Lock()
	{{camel .ModelName}}Initialised = true
	{{camel .ModelName}}MX.Unlock()
	for {
		time.Sleep(time.Millisecond * 20)

		{{camel .ModelName}}MX.Lock()
		if len({{camel .ModelName}}FRs) > 0 {
			var {{camel .ModelName}}s []pm.{{.PmName}}Full
			var err error
			var ids []string

			for _, r := range {{camel .ModelName}}FRs {
				ids = append(ids, r.{{.ModelName}}ID)
			}

			log.Printf("Batched {{camel .ModelName}} size: %d", len({{camel .ModelName}}FRs))
			{{camel .ModelName}}s, err = pm.GetMulti{{.PmName}}Full(context.Background(), l.pool, ids)

		OUTER:
			for _, r := range {{camel .ModelName}}FRs {
				for _, c := range {{camel .ModelName}}s {
					if c.{{.ModelName}}ID == r.{{.ModelName}}ID {
						r.Reply <- {{.ModelName}}FetchReply{ {{.ModelName}}: c, Error: nil}
						continue OUTER
					}
				}

				err2 := err

				if err2 == nil {
					err2 = fmt.Errorf("Not found")
				}
				r.Reply <- {{.ModelName}}FetchReply{Error: err2}
			}

			{{camel .ModelName}}FRs = []{{.ModelName}}FetchRequest{}
		}

		{{camel .ModelName}}MX.Unlock()
	}
}

// GetAll{{.ModelName}} Returns an array of all {{.ModelName}} entries, using the provided filter
func (l *Loader) GetAll{{.ModelName}}(ctx context.Context, filter models.Filter) (all []models.{{.ModelName}}, pi models.PageInfo, count int, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetAll{{.ModelName}}")
	defer span.Finish()

	descending := filter.Order.Descending
	// If filter.Before, we reverse the order of the results now:
	if filter.Before {
		filter.Order.Descending = !descending
	}

	r, hasMore, count, err := pm.QueryPaginated{{.PmName}}Full(ctx, l.pool, filter.Cursor, filter.Where, filter.Order, filter.Count)

	if err != nil {
		return
	}

	// We may need to reverse the order back again if we swapped it:
	if descending != filter.Order.Descending {
		// Restore the order
		for i := len(r)/2 - 1; i >= 0; i-- {
			opp := len(r) - 1 - i
			r[i], r[opp] = r[opp], r[i]
		}
	}

	if filter.Before {
		pi.HasPreviousPage = hasMore
		if filter.Cursor != nil {
			pi.HasNextPage = true
		}
	} else {
		pi.HasNextPage = hasMore
		if filter.Cursor != nil {
			pi.HasPreviousPage = true
		}
	}


	all = make([]models.{{.ModelName}}, len(r))
	for i, b := range r {
		all[i] = hydrateModel{{.ModelName}}(ctx, b)
	}

	return
}

{{if .Create}}
// Update{{.ModelName}} Updates {{.ModelName}} based on provided changes
func (l *Loader) Update{{.ModelName}}(ctx context.Context, id string, u map[string]interface{}) error {
	tx, err := l.pool.Begin()

	if err != nil {
		return err
	}

	err = l.update{{.ModelName}}(ctx, tx, id, u)
	if rollbackErr(err, tx) != nil {
		return err
	}

	return tx.Commit()
}

// update{{.ModelName}} Updates {{.ModelName}} based on provided changes using provided db connection
func (l *Loader) update{{.ModelName}}(ctx context.Context, db gnorm.DB, id string, u map[string]interface{}) error {
	{{camel .ModelName}}, err := pm.Get{{.PmName}}Full(ctx, l.pool, id)

	if err != nil {
		return err
	}

	// Helps us keep track of which field has any errors
	pathCtx := addPathToContext(ctx, kace.Snake("{{.ModelName}}"))

	// By iterating over the map entries, we can ensure we only modify those values that are set:
	for k, v := range u {
		err = l.update{{.ModelName}}Field(pathCtx, false, db, &{{camel .ModelName}}, k, v)

		if err != nil {
			return fmt.Errorf("%s: %s", k, err)
		}
	}

	l.validate{{.PmName}}(pathCtx, {{camel .ModelName}})

	if validate.HasErrors(ctx) {
		log.Print("Found validation errors in update{{.ModelName}}")

		// Only return an error if this is top path:
		if isTopPath(ctx) {
			log.Printf("Validation errors: %s", validate.ErrorsString(ctx))
			return fmt.Errorf("Unresolved validation errors, cannot complete action")
		}
		return nil
	}

	_, err = pm.Upsert{{.PmName}}Full(ctx, db, {{camel .ModelName}})

	return sanitiseError(err)
}

// create{{.PmName}} Creates {{.PmName}} from given input
func (l *Loader) create{{.PmName}}(ctx context.Context, db DB, i map[string]interface{}) (o pm.{{.PmName}}Full, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "create{{.PmName}}")
	defer span.Finish()

	// Helps us keep track of which field has any errors
	pathCtx := addPathToContext(ctx, kace.Snake("{{.ModelName}}"))

	for k, v := range i {
		err = l.update{{.ModelName}}Field(pathCtx, true, db, &o, k, v)

		if err != nil {
			err = fmt.Errorf("%s: %s", k, err)
			return
		}
	}

	l.validate{{.PmName}}(pathCtx, o)

	if validate.HasErrors(ctx) {
		log.Print("Found validation errors in create{{.PmName}}")
		if isTopPath(ctx) {
			log.Printf("Validation errors: %s", validate.ErrorsString(ctx))
			return o, fmt.Errorf("Unresolved validation errors, cannot complete action")
		}
		return o, nil
	}

	o, err = pm.Upsert{{.PmName}}Full(ctx, db, o)

	return o, sanitiseError(err)
}
{{end}}

`))

var linkTemplate = template.Must(template.New("").Funcs(templateFuncs).Parse(
	`// Code generated by go generate; DO NOT EDIT.
// This file was generated by robots
package postgres

import (
	"bitbucket.org/replaceme/api/loaders"
	"bitbucket.org/replaceme/gnorm/gnorm"
	"bitbucket.org/replaceme/gnorm/gnorm/pm"
	"bitbucket.org/replaceme/api/models"
	"bitbucket.org/replaceme/api/opa"
	opentracing "github.com/opentracing/opentracing-go"
)

{{ range $x, $c :=  .Models -}}
{{$m1 := $c.Model1}}
{{$m2 := $c.Model2}}
{{$full := concat $m1 $m2}}
// Link{{$c.Model1}}{{$c.Model2}} Links '{{$c.Model1}}' to {{$c.Model2}}'
func (l *Loader) Link{{$c.Model1}}{{$c.Model2}}(ctx context.Context, {{camel $m1}}ID string, {{camel $m2}}ID string, link bool) (bool, error) {
	if link {
		var clink pm.{{$m1}}{{$m2}}

		clink.{{$c.Model1}}ID{{$c.Model1}} = {{camel $c.Model1}}ID
		clink.{{$c.Model2}}ID{{$c.Model2}} = {{camel $c.Model2}}ID

		_, err := pm.Upsert{{$m1}}{{$m2}}(ctx, l.pool, clink)

		// Sanitise our output, and log errors if needed:
		err = sanitiseError(err)

		return (err == nil), err
	}

	// !link, therefore delete any such connection:
	res, err := l.pool.Exec("DELETE FROM pm.{{snake $full}} WHERE {{snake $c.Model1}}_id_{{snake $c.Model1}}=$1 AND {{snake $c.Model2}}_id_{{snake $c.Model2}}=$2", {{camel $m1}}ID, {{camel $m2}}ID)

	err = sanitiseError(err)
	if err == nil && res.RowsAffected() == 0 {
		err = fmt.Errorf("No such link exists")
	}
	return (err == nil), err
}
{{end}}
`))

var resolverTemplate = template.Must(template.New("").Funcs(templateFuncs).Parse(
	`// Code generated by go generate; DO NOT EDIT.
// This file was generated by robots
package resolvers

import (
	"bitbucket.org/replaceme/api/loaders"
	"bitbucket.org/replaceme/api/models"
	"bitbucket.org/replaceme/api/opa"
	"bitbucket.org/replaceme/gnorm/gnorm"
	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/gqlerror"
	opentracing "github.com/opentracing/opentracing-go"
)

func (r *queryResolver) EditableUpdate{{.ModelName}}Fields(ctx context.Context, id string) ([]string, error) {
	return editableUpdate{{.ModelName}}Fields(ctx, id)
}

{{if .Update}}
// Update{{.ModelName}} Updates {{.ModelName}} with provided input
func (r *mutationResolver) Update{{.ModelName}}(ctx context.Context, id string, u map[string]interface{}) (*models.{{.ModelName}}, error) {
	// Get allowed edit fields:
	allowed, err := editableUpdate{{.ModelName}}Fields(ctx, id)
	if err != nil {
		return nil, err
	}

	// Filter out unapproved changes
	changes, _, any, err := authorisedChanges(ctx, allowed, u)

	if err != nil {
		return nil, err
	}

	if !any {
		return nil, fmt.Errorf("No fields were permitted to be updated")
	}

	err = loaders.Current.Update{{.ModelName}}(ctx, id, changes)

	if err != nil {
		return nil, err
	}

	obj, err := loaders.Current.Get{{.ModelName}}(ctx, id)

	return &obj, err
}
{{end}}
{{if .PrepareCreate}}
// prepareCreate{{.ModelName}} Performs some pre-processing on the provided map.  Disable generation of this function and create your own if you require any pre-processing performed
func prepareCreate{{.ModelName}}(ctx context.Context, i map[string]interface{}) error {
	return nil
}
{{end}}
{{if .Create}}
// Create{{.ModelName}} Creates {{.ModelName}}
func (r *mutationResolver) Create{{.ModelName}}(ctx context.Context, i map[string]interface{}) (*models.{{.ModelName}}, error) {
	err := prepareCreate{{.ModelName}}(ctx, i)
	if err != nil {
		return nil, err
	}

	id, err := loaders.Current.Create{{.ModelName}}(ctx, i)

	if err != nil {
		return nil, err
	}

	obj, err := loaders.Current.Get{{.ModelName}}(ctx, id)
	return &obj, err
}
{{end}}
{{if .Query}}
func query{{.PluralModelName}}(ctx context.Context, first *int, after *string, last *int, before *string, cf *models.{{.ModelName}}Filter, sortField *models.{{.ModelName}}Sort, sortDirection *models.SortDirection, where []gnorm.WhereClause) (o models.{{.PluralModelName}}Connection, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "query{{.PluralModelName}}")
	defer span.Finish()
//	limit, err := opa.GetInt(ctx, "data.api.limits.{{camel .PluralModelName}}ConnectionLimit", map[string]interface{}{})
//
//	if err != nil {
//		return
//	}
//
//	if (first != nil && int64(*first) > limit) || (last != nil && int64(*last) > limit) {
//		err = fmt.Errorf("Cannot request more than %d entries for query", limit)
//		return
//	}
//
//
//	// Use the policy defined base amount if none provided
//	if (first == nil) && (last == nil) {
//		var baseLimit int64
//		baseLimit, err = opa.GetInt(ctx, "data.api.limits.{{camel .PluralModelName}}ConnectionCount", map[string]interface{}{})
//
//		if err != nil {
//			return
//		}
//
//		intLimit := int(baseLimit)
//		first = &intLimit
//		last = &intLimit
//	}

	f := models.NewFilter(first, after, last, before, sortDirection)

	// Set up the sort order based on inputs:
	if sortField != nil {
		var err error

		f.Order, err = sort{{.ModelName}}(ctx, *sortField, f.Order)

		if err != nil {
			return o, fmt.Errorf("Cannot sort by field %s: %s", sortField, err)
		}
	}

	// Configure the where clauses:
	if cf != nil {
		var fw []gnorm.WhereClause
		fw, err = filter{{.ModelName}}(ctx, *cf)
		if err != nil {
			return
		}

		where = append(where, fw...)
	}

	f.Where = where

	r, pi, count, err := loaders.Current.GetAll{{.ModelName}}(ctx, f)

	if err != nil {
		return o, err
	}

	o.PageInfo = pi
	o.TotalCount = count
	o.Edges = make([]models.{{.ModelName}}Edge, len(r))

	for i, t := range r {
		o.Edges[i] = models.{{.ModelName}}Edge{Cursor: t.ID, Node: t}
	}

	return o, err
}
{{end}}
`))
