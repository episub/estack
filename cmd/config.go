package cmd

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// Config Store values read from config.yaml file
type Config struct {
	PackageName string   `yaml:"packageName"`
	Generate    Generate `yaml:"generate"`
}

// Generate Stores generate values from yaml
type Generate struct {
	Resolvers []ResolverGenerate `yaml:"resolvers"`
	Postgres  []PostgresGenerate `yaml:"postgres"`
}

// ResolverGenerate Which resolver related things to generate code for
type ResolverGenerate struct {
	SingularModelName string `yaml:"singularName"`
	PluralModelName   string `yaml:"pluralName"`
	Create            bool   `yaml:"create"`        // Build a create function
	Update            bool   `yaml:"update"`        // Build an update function
	PrepareCreate     bool   `yaml:"prepareCreate"` // Provide a prepare function for you (set to false if you want to set one yourself)
	Query             bool   `yaml:"query"`         // Creates a queryX function used for pagination via a connections type method
}

// PostgresGenerate Which postgres helper functions to generate code for
type PostgresGenerate struct {
	ModelName         string `yaml:"modelName"`         // Name of model used by GraphQL
	ModelPackage      string `yaml:"modelPackage"`      // Name of the package containing the model
	ModelPackageShort string `yaml:"modelPackageShort"` // Last part of the package name
	PmName            string `yaml:"postgresName"`      // Name of postgres data object
	PK                string `yaml:"primaryKey"`        // Go struct for database name for primary key field
	PrimaryKeyType    string `yaml:"primaryKeyType"`    // Go type for primary key
	Create            bool   `yaml:"create"`            // Generate create/update related functions
}

func readConfig(filename string) (Config, error) {
	input, err := ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = yaml.Unmarshal(input, &config)
	if err != nil {
		return config, err
	}

	return config, err
}
