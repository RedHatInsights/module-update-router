package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/peterbourgon/ff/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

func main() {
	var (
		addr       string // addr is the TCP address and port the application listens on
		maddr      string // maddr is the TCP address and port the metrics HTTP server listens on
		logLevel   string // log level
		env        string // env determines operation mode (log formatters, etc.)
		dbdata     string // initial data to populate database
		pathprefix string // API path prefix
		appname    string // API path app name
		dbHost     string // IP or hostname of database server
		dbPort     string // TCP port on database server
		dbName     string // database name
		dbUser     string // database username
		dbPass     string // database user password
	)

	const (
		apiversion = "v1"
	)

	fs := flag.NewFlagSet("module-update-router", flag.ExitOnError)
	fs.StringVar(&addr, "addr", ":8080", "app listen address")
	fs.StringVar(&maddr, "maddr", ":2112", "metrics listen address")
	fs.StringVar(&logLevel, "log-level", "info", "default logging level")
	fs.StringVar(&env, "environment", "development", "operation mode")
	fs.StringVar(&pathprefix, "path-prefix", "/api", "API path prefix")
	fs.StringVar(&appname, "app-name", "", "name component for the API prefix")
	fs.StringVar(&dbdata, "db-data", "", "initial database seed data")
	fs.StringVar(&dbHost, "db-host", "localhost", "IP or hostname of database")
	fs.StringVar(&dbPort, "db-port", "5432", "TCP port on database server")
	fs.StringVar(&dbName, "db-name", "postgres", "database name")
	fs.StringVar(&dbUser, "db-user", "postgres", "database username")
	fs.StringVar(&dbPass, "db-pass", "", "database user password")

	ff.Parse(fs, os.Args[1:], ff.WithEnvVarNoPrefix())

	switch env {
	case "production":
		log.SetFormatter(&log.JSONFormatter{})
	default:
		log.SetFormatter(&log.TextFormatter{})
	}

	lvl, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(lvl)

	var driver, connString string
	switch env {
	case "production":
		driver = "pgx"
		connString = fmt.Sprintf("host=%s port=%s database=%s user=%s password=%s",
			dbHost, dbPort, dbName, dbUser, dbPass)
	default:
		driver = "sqlite3"
		connString = "file::memory:?cache=shared"

		if envDriver, ok := os.LookupEnv("DB_DRIVER"); ok {
			driver = envDriver
			connString = dbName
		}
	}

	db, err := Open(driver, connString)
	if err != nil {
		log.Fatal(err)
	}

	if len(dbdata) > 0 {
		if err := db.Load(dbdata); err != nil {
			log.Fatal(err)
		}
	}

	srv, err := NewServer(addr, path.Join(pathprefix, appname, apiversion), db)
	if err != nil {
		log.Fatal(err)
	}
	defer srv.Close()

	go func() {
		log.WithFields(log.Fields{
			"func": "metrics",
			"addr": maddr,
		}).Info("started")
		http.ListenAndServe(maddr, promhttp.Handler())
	}()

	go func() {
		log.WithFields(log.Fields{
			"func": "app",
			"addr": addr,
		}).Info("started")
		log.Fatal(srv.ListenAndServe())
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit
}
