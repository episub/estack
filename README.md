estack is used to initialise and manage a project for Episub's stack.  For now, this involves the setting up of a repository to handle the server side of things.

This framework makes use of these projects (among others) to auto-generate some key code and provide some key functionality:

* Gnorm for database inspection and code generation
* gqlgen for graphql code generation
* Open Policy Agent for permissions

This framework is designed to allow you to use as much or as little as you want.  We provide tools that generate useful functions for you, but you can easily turn that code off if you want and use your own custom code.

# TODO
  
* Batched queries:
 - Don't have db as a parameter when it's not even used
 - Have a non-batched option that can be used for transactions
* Update all return values in `cmd/static/loader/gen.gotmpl`  to return sanitised errors
* have config setting to specify copying of templates every gen time
* Allow modification of all the template files, like with gnorm
* Instead of map, use a structural, with a string array naming the fields that are provided or to be updated, to allow us to distinguish between 'no change' vs 'null this field'.
* For inherited tables, have gnorm generate code that reuses objects from the parent table in child tables.  This will make it easier to reuse tests, hydration, etc, instead of duplicating
* Fix gqlgen and gnorm versions within estack somehow

# Initialise Project

This project relies on using Go modules, so you will need Go 1.11 or higher installed, and use a folder outside of $GOPATH.  Initialisation should be run only once.  It will set up a new project with fresh configs.  Create a folder for your new repository, and initialise your modules file:

```
go mod init github.com/example/project
```

Now we're ready to generate our project (optionally specify --folder=<folder>):

```
go run github.com/episub/estack init --package=github.com/example/project
```

Your base project is now ready, including a sample SQL file for PostgreSQL in the migrations folder, and a schema for GraphQL in schema.graphql.  Let's use the base project.  The key to the Episub stack is auto generated code.  When changes are made to key files, we must re-generate our code.

Before we can do this, we need the database running so that we can connect to the database and create the relevant DB code:

```
docker-compose up -d
```

Now we are ready to generate our code:

```
go run scripts/estack.go generate
```

This gives us the GraphQL code.  Open up `resolvers/resolver.go`.  There is a bug at the moment, so we need to add the import `"github.com/episub/stacktest/graph/generated"` at the top, and ensure that MutationResolver and QueryResolver use this package.  i.e., `generated.MutationResolver` and `generated.QueryResolver`.

That's it.  Now you're ready to run the project:

```
go run server.go
```

Connect to the project via http://localhost:8080 and try the following query:

```
query {
  todos {
    id
    content
  }
}
```

You will see an error results, because we haven't implemented this resolver yet.  In fact, the default code has a panic hard coded into it.  Fill out the query with something like the following:

```
func (r *queryResolver) Todos(ctx context.Context) ([]models.Todo, error) {
	todos := make([]models.Todo, 3)

	for i, _ := range todos {
		todos[i] = models.Todo{
			ID:   fmt.Sprintf("%d", i),
			Content: fmt.Sprintf("Todo number %d", i),
		}
	}

	return todos, nil
}
```

Start the server back up again, and try the query again.  You should now see it working!

# Database

This project by default separates database functions (gnorm/dbl folder) from GraphQL models.  The best database designs do not necessarily describe or carve up the world in the way that makes sense for your GraphQL API.  These are separate ways of representing the world, and it may in some cases be useful to keep them separate.  For example, while your database may be fully normalised, you may want the GraphQL to display a de-normalised model.  Furthermore, you may decide one day that you want to switch out your storage solution, and it will help to not have the resolvers tightly coupled to the storage system.  So we prefer to use the 'loader' package to contain functions that don't leak the database to the resolvers, and interact through that.  The job of the 'loader' package is to translate between the GraphQL model of the world and our database's model of the world.

Let's pull our todos from the database rather than hard coding the reply.  Update your config to set it to auto-generate some query related functions.

Update your config.yaml to the following:

```
packageName: "github.com/episub/stacktest"
generate:
  postgres:
  - modelName: "Todo"
    postgresName: "Todo"
    primaryKey: "TodoID"
    primaryKeyType: "int"
```

We now need to  manually create some more entries in order to use the provided auto-generated loader functions.  When we want to use a relay connections type response, we need a PageInfo object.  We won't be using it yet, but the auto-generated functions require it.  So let's add it to our schema.graphql file:

Finally, because we're separating the database representation of our data from the GraphQL API, we need to perform the translation.  In many cases we'll find ourselves using models that happen to match the database layout very closely, but this may not always be so.  Create the file `loader/todo.go` with the following content:

```
package loader

import (
	"context"
	"fmt"

	"github.com/episub/stacktest/gnorm/dbl"
	"github.com/episub/stacktest/models"
)

func hydrateModelTodo(ctx context.Context, i dbl.TodoFull) (o models.Todo) {
	o.ID = fmt.Sprintf("%d", i.TodoID)
	o.Content = i.Content
	o.Done = i.Done

	return
}
```

In `loader/init.go`, update the runBatchLoaders() function with the one for our new function.  This lets us batch together some requests to reduce database calls:

```
	go Loader.runTodoBatcher()
```

We also need to ensure the database is initialised.  Open up `server.go` and let's initialise the loader by adding the following early in the main() function (after env.Parse):

```
	err = loader.InitialiseLoader(cfg.DBName, cfg.DBUser, cfg.DBPass, cfg.DBHost, log)
	if err != nil {
		log.Fatal(err)
	}
```

And now for the main part, where we update the resolver to return the loader returned values.  In `resolvers/resolver.go`, update our resolver function to be as follows:

```
func (r *queryResolver) Todos(ctx context.Context) ([]models.Todo, error) {
	all, _, _, err := loader.Loader.GetAllTodo(ctx, models.Filter{})
	return all, err
}
```

Finally, we re-run the generated and then run the server.  We pass in configuration values as environment variables so that it's easier to set for docker based production deployments:

```
go run scripts/estack.go generate
DB_USER=estack DB_PASS=estack go run server.go
```

Try your query again, and it should now return the results from the database!

It may seem more cumbersome to have to translate between the database and GraphQL, but this extra burden comes with the benefit of clear separation of concerns that should be separate.  It allows us to break the symmetry between database model and GraphQL model where needed, and allows a much more flexible design.

## User

The above works, but we won't be able to fetch the user, and being able to traverse the graph to do this is one of the main benefits of GraphQL.  To implement this requires us to override the auto-generated models so that we can store the user ID rather than the full user (which currently does not get fetched from the database).  In models/generated.go we can find the auto-created Todo struct.  Move that into its own file `models/todo.go`, and change User to UserID string:

```
package models

type Todo struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Done    bool   `json:"done"`
	UserID  string
}
```

Update `gqlgen.yml` at the bottom with the following to tell gqlgen that we are going to provide the Todo model ourselves:

```
models:
  Todo:
    model: github.com/episub/stacktest/models.Todo
```

Update `config.yaml` to generate user related functions:

```
packageName: "github.com/episub/stacktest"
generate:
  postgres:
  - modelName: "Todo"
    postgresName: "Todo"
    primaryKey: "TodoID"
    primaryKeyType: "int"
  - modelName: "User"
    postgresName: "User"
    primaryKey: "UserID"
    primaryKeyType: "int"
```

As we did above, we need to create a hydrate function to translate from a database user to a GraphQL user.  Create the file `loader/user.go` with the following contents:

```
package loader

import (
	"context"
	"fmt"

	"github.com/episub/stacktest/gnorm/dbl"
	"github.com/episub/stacktest/models"
)

func hydrateModelUser(ctx context.Context, i dbl.UserFull) (o models.User) {
	o.ID = fmt.Sprintf("%d", i.UserID)
	o.Username = i.Username
	o.Admin = i.Admin

	return
}
```

Update hydrateModelTodo in `loader/todo.go` to add a line setting the User ID:

```
	o.UserID = fmt.Sprintf("%d", i.UserID)
```

And add the batch loader in `loader/init.go`:

```
	go Loader.runUserBatcher()
```

If you regenerate now and try to run the server, you'll see an error:
```
go run scripts/estack.go generate
DB_USER=estack DB_PASS=estack go run server.go
```

This error is telling us that we no longer implement the Todo method.  We need to provide our own Todo resolver, and along with that a User resolver since Todo no longer has a User field that gqlgen can use to return the value.  Update `resolvers/resolver.go`  by adding the following:

```
func (r *Resolver) Todo() generated.TodoResolver {
	return &todoResolver{r}
}

func (t *todoResolver) User(ctx context.Context, obj *models.Todo) (models.User, error) {
	i, err := strconv.ParseInt(obj.UserID, 10, 64)
	if err != nil {
		return models.User{}, fmt.Errorf("Invalid format for user ID.  Must be an integer")
	}
	return loader.Loader.GetUser(ctx, int(i))
}
```

Return to the GraphQL explorer and try your new  query out:

```
query {
  todos {
    id
    content
    user {
      id
      username
      admin
    }
  }
}
```

And see the results!

# Pagination

This project provides some useful tools for auto-generating code to allow for pagination.  Suppose you wanted to provide a paginated list for todos.  Update your config to look like this:

```
packageName: "github.com/episub/stacktest"
generate:
  resolvers:
  - singularName: "todo"
    pluralName: "todos"
    query: true
  postgres:
  - modelName: "Todo"
    postgresName: "Todo"
    primaryKey: "TodoID"
```

# User Permissions

This project makes use of Open Policy Agent to give a powerful and highly flexible permissions framework.

# PostgreSQL Advice

* Use audit tables for storing and tracking history
