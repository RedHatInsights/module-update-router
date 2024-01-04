package config

import (
	"flag"
	"fmt"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/sgreben/flagvar"
)

// Config stores values that are used to configure the application.
type Config struct {
	Addr         string
	APIVersion   string
	AppName      string
	EventBuffer  int
	LogFormat    flagvar.Enum
	LogLevel     string
	MAddr        string
	MetricsTopic string
	PathPrefix   string
	Reset        bool
	SeedPath     flagvar.File
}

// DefaultConfig is the default configuration variable, providing access to
// configuration values globally.
var DefaultConfig Config = Config{
	Addr:         ":8080",
	APIVersion:   "v1",
	AppName:      "module-update-router",
	EventBuffer:  1000,
	LogFormat:    flagvar.Enum{Choices: []string{"text", "json"}, Value: "text"},
	LogLevel:     "info",
	MAddr:        ":2112",
	MetricsTopic: "client-metrics",
	PathPrefix:   "/api",
	Reset:        false,
	SeedPath:     flagvar.File{},
}

// init can be used to set default values for DefaultConfig that require more
// complex computation, such as external package function calls.
func init() {
	if clowder.IsClowderEnabled() {
		DefaultConfig.Addr = fmt.Sprintf(":%v", *clowder.LoadedConfig.PublicPort)
		DefaultConfig.MAddr = fmt.Sprintf(":%v", clowder.LoadedConfig.MetricsPort)
	}
}

// FlagSet creates a new FlagSet, defined with flags for each struct field in
// the DefaultConfig variable.
func FlagSet(name string, errorHandling flag.ErrorHandling) *flag.FlagSet {
	fs := flag.NewFlagSet(name, errorHandling)

	fs.Var(&DefaultConfig.LogFormat, "log-format", fmt.Sprintf("set logging format (%v)", DefaultConfig.LogFormat.Help()))
	fs.StringVar(&DefaultConfig.LogLevel, "log-level", DefaultConfig.LogLevel, "logging level")
	fs.Var(&DefaultConfig.SeedPath, "seed-path", "path to the SQL seed file")
	fs.BoolVar(&DefaultConfig.Reset, "reset", DefaultConfig.Reset, "drop all tables before running migrations")
	fs.StringVar(&DefaultConfig.Addr, "addr", DefaultConfig.Addr, "app listen address")
	fs.StringVar(&DefaultConfig.APIVersion, "api-version", DefaultConfig.APIVersion, "version to use in the URL path")
	fs.StringVar(&DefaultConfig.AppName, "app-name", DefaultConfig.AppName, "name component for the API prefix")
	fs.IntVar(&DefaultConfig.EventBuffer, "event-buffer", DefaultConfig.EventBuffer, "the size of the event channel buffer")
	fs.StringVar(&DefaultConfig.MAddr, "maddr", DefaultConfig.MAddr, "metrics listen address")
	fs.StringVar(&DefaultConfig.MetricsTopic, "metrics-topic", DefaultConfig.MetricsTopic, "topic on which to place metrics data")
	fs.StringVar(&DefaultConfig.PathPrefix, "path-prefix", DefaultConfig.PathPrefix, "API path prefix")

	return fs
}
