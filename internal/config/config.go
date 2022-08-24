package config

import (
	"flag"
	"fmt"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
)

// Config stores values that are used to configure the application.
type Config struct {
	Addr           string
	AppName        string
	DBDriver       string
	DBHost         string
	DBName         string
	DBPass         string
	DBPort         int
	DBURL          string
	DBUser         string
	EventBuffer    int
	KafkaBootstrap string
	LogFormat      string
	LogLevel       string
	MAddr          string
	MetricsTopic   string
	Migrate        bool
	PathPrefix     string
	Reset          bool
	SeedPath       string
}

// DefaultConfig is the default configuration variable, providing access to
// configuration values globally.
var DefaultConfig Config = Config{
	Addr:           ":8080",
	AppName:        "",
	DBDriver:       "sqlite3",
	DBHost:         "localhost",
	DBName:         "postgres",
	DBPass:         "",
	DBPort:         5432,
	DBURL:          "",
	DBUser:         "postgres",
	EventBuffer:    1000,
	KafkaBootstrap: "",
	LogFormat:      "text",
	LogLevel:       "info",
	MAddr:          ":2112",
	MetricsTopic:   "client-metrics",
	Migrate:        false,
	PathPrefix:     "/api",
	Reset:          false,
	SeedPath:       "",
}

// init can be used to set default values for DefaultConfig that require more
// complex computation, such as external package function calls.
func init() {
	if clowder.IsClowderEnabled() {
		DefaultConfig.Addr = fmt.Sprintf(":%v", *clowder.LoadedConfig.PublicPort)
		DefaultConfig.DBHost = clowder.LoadedConfig.Database.Hostname
		DefaultConfig.DBName = clowder.LoadedConfig.Database.Name
		DefaultConfig.DBPass = clowder.LoadedConfig.Database.Password
		DefaultConfig.DBPort = clowder.LoadedConfig.Database.Port
		DefaultConfig.DBUser = clowder.LoadedConfig.Database.Username
		DefaultConfig.MAddr = fmt.Sprintf(":%v", clowder.LoadedConfig.MetricsPort)
	}
}

// FlagSet creates a new FlagSet, defined with flags for each struct field in
// the DefaultConfig variable.
func FlagSet(name string, errorHandling flag.ErrorHandling) *flag.FlagSet {
	fs := flag.NewFlagSet(name, errorHandling)

	fs.StringVar(&DefaultConfig.Addr, "addr", DefaultConfig.Addr, "app listen address")
	fs.StringVar(&DefaultConfig.MAddr, "maddr", DefaultConfig.MAddr, "metrics listen address")
	fs.StringVar(&DefaultConfig.LogLevel, "log-level", DefaultConfig.LogLevel, "logging level")
	fs.StringVar(&DefaultConfig.LogFormat, "log-format", DefaultConfig.LogFormat, "set logging format (choice of 'json' or 'text')")
	fs.StringVar(&DefaultConfig.PathPrefix, "path-prefix", DefaultConfig.PathPrefix, "API path prefix")
	fs.StringVar(&DefaultConfig.AppName, "app-name", DefaultConfig.AppName, "name component for the API prefix")
	fs.StringVar(&DefaultConfig.DBDriver, "db-driver", DefaultConfig.DBDriver, "database driver ('pgx' or 'sqlite3')")
	fs.StringVar(&DefaultConfig.DBHost, "db-host", DefaultConfig.DBHost, "IP or hostname of database server")
	fs.IntVar(&DefaultConfig.DBPort, "db-port", DefaultConfig.DBPort, "TCP port on database server")
	fs.StringVar(&DefaultConfig.DBName, "db-name", DefaultConfig.DBName, "database name")
	fs.StringVar(&DefaultConfig.DBUser, "db-user", DefaultConfig.DBUser, "database username")
	fs.StringVar(&DefaultConfig.DBPass, "db-pass", DefaultConfig.DBPass, "database user password")
	fs.StringVar(&DefaultConfig.DBUser, "database-url", DefaultConfig.DBURL, "database connection URL")
	fs.StringVar(&DefaultConfig.MetricsTopic, "metrics-topic", DefaultConfig.MetricsTopic, "topic on which to place metrics data")
	fs.StringVar(&DefaultConfig.KafkaBootstrap, "kafka-bootstrap", DefaultConfig.KafkaBootstrap, "url of the kafka broker for the cluster")
	fs.IntVar(&DefaultConfig.EventBuffer, "event-buffer", DefaultConfig.EventBuffer, "the size of the event channel buffer")
	fs.BoolVar(&DefaultConfig.Migrate, "migrate", DefaultConfig.Migrate, "run migrations")
	fs.StringVar(&DefaultConfig.SeedPath, "seed-path", DefaultConfig.SeedPath, "path to the SQL seed file")
	fs.BoolVar(&DefaultConfig.Reset, "reset", DefaultConfig.Reset, "drop all tables before running migrations")

	return fs
}
