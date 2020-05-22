package main

import (
	"flag"
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
		logLevel   string // log level
		maddr      string // maddr is the TCP address and port the metrics HTTP server listens on
		dbpath     string // dbpath is a file path to the database
		env        string // env determines operation mode (log formatters, etc.)
		dbdata     string // initial data to populate database
		pathprefix string // API path prefix
		appname    string // API path app name
	)

	const (
		apiversion = "v1"
	)

	fs := flag.NewFlagSet("module-update-router", flag.ExitOnError)
	fs.StringVar(&logLevel, "log-level", "info", "default logging level")
	fs.StringVar(&addr, "addr", ":8080", "app listen address")
	fs.StringVar(&maddr, "maddr", ":2112", "metrics listen addr")
	fs.StringVar(&dbpath, "db-path", "file::memory:?cache=shared", "path to database")
	fs.StringVar(&env, "environment", "development", "operation mode")
	fs.StringVar(&dbdata, "db-data", "", "initial database seed data")
	fs.StringVar(&pathprefix, "path-prefix", "/api", "API path prefix")
	fs.StringVar(&appname, "app-name", "", "name component for the API prefix")

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

	srv, err := NewServer(addr, dbpath, path.Join(pathprefix, appname, apiversion))
	if err != nil {
		log.Fatal(err)
	}
	defer srv.Close()

	if len(dbdata) > 0 {
		if err := srv.db.Load(dbdata); err != nil {
			log.Fatal(err)
		}
	}

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
