package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/peterbourgon/ff/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

func main() {
	var (
		addr       string
		maddr      string
		logLevel   string
		logFormat  string
		pathprefix string
		appname    string
		dbDriver   string
		dbHost     string
		dbPort     string
		dbName     string
		dbUser     string
		dbPass     string
		dbURL      string
		migrate    bool
		seedpath   string
		reset      bool
	)

	const (
		apiversion = "v1"
	)

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&addr, "addr", ":8080", "app listen address")
	fs.StringVar(&maddr, "maddr", ":2112", "metrics listen address")
	fs.StringVar(&logLevel, "log-level", "info", "logging level")
	fs.StringVar(&logFormat, "log-format", "text", "set logging format (choice of 'json' or 'text')")
	fs.StringVar(&pathprefix, "path-prefix", "/api", "API path prefix")
	fs.StringVar(&appname, "app-name", "", "name component for the API prefix")
	fs.StringVar(&dbDriver, "db-driver", "sqlite3", "database driver ('pgx' or 'sqlite3')")
	fs.StringVar(&dbHost, "db-host", "localhost", "IP or hostname of database server")
	fs.StringVar(&dbPort, "db-port", "5432", "TCP port on database server")
	fs.StringVar(&dbName, "db-name", "postgres", "database name")
	fs.StringVar(&dbUser, "db-user", "postgres", "database username")
	fs.StringVar(&dbPass, "db-pass", "", "database user password")
	fs.StringVar(&dbURL, "database-url", "", "database connection URL")
	fs.BoolVar(&migrate, "migrate", false, "run migrations")
	fs.StringVar(&seedpath, "seed-path", "", "path to the SQL seed file")
	fs.BoolVar(&reset, "reset", false, "drop all tables before running migrations")

	ff.Parse(fs, os.Args[1:], ff.WithEnvVarNoPrefix())

	if dbURL == "" && (dbHost == "" || dbPort == "" || dbName == "" || dbUser == "") {
		log.Fatal("error: unable to connect to database. See -help for details")
	}

	switch logFormat {
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	default:
		log.SetFormatter(&log.TextFormatter{})
	}

	lvl, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(lvl)

	var connString string
	switch dbDriver {
	case "pgx":
		if dbURL != "" {
			connString = dbURL
		} else {
			connString = fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
				dbUser, dbPass, dbHost, dbPort, dbName)
		}
	case "sqlite3":
		if dbURL != "" {
			connString = dbURL
		} else {
			connString = "file::memory:?cache=shared"
		}
	default:
		log.Fatalf("error: unsupported database: %v", dbDriver)
	}

	db, err := Open(dbDriver, connString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if migrate {
		log.Debug("running migrations")
		if err := db.Migrate(reset); err != nil {
			log.Fatal(err)
		}
		log.Debug("migrations complete")

		if seedpath != "" {
			log.Debug("seeding database")
			if err := db.Seed(seedpath); err != nil {
				log.Fatal(err)
			}
			log.Debug("seed complete")
		}
		os.Exit(0)
	}

	apiroots := strings.Split(pathprefix, ",")
	for i, root := range apiroots {
		apiroots[i] = path.Join(root, appname, apiversion)
	}

	srv, err := NewServer(addr, apiroots, db)
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
