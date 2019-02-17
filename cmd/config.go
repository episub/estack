package cmd

// Config Store values read from config.yml file
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
	ModelName string `yaml:"modelName"`    // Name of model used by GraphQL
	PmName    string `yaml:"postgresName"` // Name of postgres data object
	PK        string `yaml:"primaryKey"`   // Go struct for database name for primary key field
	Create    bool   `yaml:"create"`       // Generate create/update related functions
}
