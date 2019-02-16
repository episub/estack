package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/99designs/gqlgen/codegen"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var gqlConfigDefault = `
schema: schema.graphql
exec:
  filename: graph/generated/generated.go
  package: generated
model:
  filename: models/generated.go
  package: models
resolver:
  filename: resolvers/resolver.go
  package: resolvers
  type: Resolver
`

var gqlSchemaDefault = `
# GraphQL schema example
#
# https://gqlgen.com/getting-started/
type Todo {
  id: ID!
  text: String!
  done: Boolean!
  user: User!
}
type User {
  id: ID!
  name: String!
}
type Query {
  todos: [Todo!]!
}
input NewTodo {
  text: String!
  userId: String!
}
type Mutation {
  createTodo(input: NewTodo!): Todo!
}
`

var dockerComposeDefault = `
version: '3'
services:
  postgres:
    image: postgres:9.6
    restart: always
    environment:
      POSTGRES_USER: estack
      POSTGRES_PASSWORD: estack
    volumes:
      - ./migrations:/docker-entrypoint-initdb.d
`

var sqlDefault = `
CREATE TABLE account(
 user_id serial PRIMARY KEY,
 username VARCHAR (50) UNIQUE NOT NULL,
 password VARCHAR (50) NOT NULL,
 email VARCHAR (355) UNIQUE NOT NULL,
 created_on TIMESTAMP NOT NULL,
 last_login TIMESTAMP
);
`

var initCmd = cli.Command{
	Name:  "init",
	Usage: "create a new estack project",
	Flags: []cli.Flag{},
	Action: func(ctx *cli.Context) {
		createFile("schema.graphql", gqlSchemaDefault)
		createFile("gqlgen.yml", gqlConfigDefault)
		createFile("docker-compose.yml", dockerComposeDefault)
		_ = os.Mkdir("migrations", 0755)
		createFile("migrations/001-base.sql", sqlDefault)
		config := generateGQL()
		codegen.GenerateServer(*config, "server.go")
	},
}

func createFile(filename string, contents string) {
	_, err := os.Stat(filename)
	if !os.IsNotExist(err) {
		return
	}

	err = ioutil.WriteFile(filename, []byte(strings.TrimSpace(contents)), 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("unable to write file %s: %s", filename, err))
		os.Exit(1)
	}
}

func generateGQL() *codegen.Config {
	var config *codegen.Config
	var err error

	config, err = codegen.LoadConfigFromDefaultLocations()
	if os.IsNotExist(errors.Cause(err)) {
		config = codegen.DefaultConfig()
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	for _, filename := range config.SchemaFilename {
		var schemaRaw []byte
		schemaRaw, err = ioutil.ReadFile(filename)
		if err != nil {
			fmt.Fprintln(os.Stderr, "unable to open schema: "+err.Error())
			os.Exit(1)
		}
		config.SchemaStr[filename] = string(schemaRaw)
	}

	if err = config.Check(); err != nil {
		fmt.Fprintln(os.Stderr, "invalid config format: "+err.Error())
		os.Exit(1)
	}

	err = codegen.Generate(*config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}

	return config
}
