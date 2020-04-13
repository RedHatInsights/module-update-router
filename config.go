package main

// Config stores settings specific to the application.
type Config struct {
	// Addr is the TCP address and port the application listens on.
	Addr string

	// Maddr is the TCP address and port the metrics HTTP server listens on.
	Maddr string

	// DBPath is a file path to the database.
	DBPath string

	// Environment determines operation mode (log formatters, etc.)
	Environment string
}

// DefaultConfig creates a new Config instance, populated with default values.
// The values are overridden by environment variable values if present.
//
// - Config.Addr (ADDR): defaults to ":8080"
//
// - Config.DBPath (DB_PATH): defaults to ":memory:"
func DefaultConfig() Config {
	return Config{
		Addr:        DefaultEnv("ADDR", ":8080"),
		Maddr:       DefaultEnv("MADDR", ":2112"),
		DBPath:      DefaultEnv("DB_PATH", "file::memory:?cache=shared"),
		Environment: DefaultEnv("ENV", "development"),
	}
}
