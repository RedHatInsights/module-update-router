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
	"time"

	"github.com/peterbourgon/ff/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redhatinsights/module-update-router/internal/config"
	log "github.com/sirupsen/logrus"
)

func main() {
	fs := config.FlagSet(os.Args[0], flag.ExitOnError)

	if err := ff.Parse(fs, os.Args[1:], ff.WithEnvVarNoPrefix()); err != nil {
		log.Fatalf("error: failed to parse flags: %v", err)
	}

	if config.DefaultConfig.DBURL == "" && (config.DefaultConfig.DBHost == "" || config.DefaultConfig.DBPort == 0 || config.DefaultConfig.DBName == "" || config.DefaultConfig.DBUser == "") {
		log.Fatal("error: unable to connect to database. See -help for details")
	}

	switch config.DefaultConfig.LogFormat {
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	default:
		log.SetFormatter(&log.TextFormatter{})
	}

	lvl, err := log.ParseLevel(config.DefaultConfig.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(lvl)

	log.SetReportCaller(true)

	log.Debugf("%+v", config.DefaultConfig)
	var connString string
	switch config.DefaultConfig.DBDriver {
	case "pgx":
		if config.DefaultConfig.DBURL != "" {
			connString = config.DefaultConfig.DBURL
		} else {
			connString = fmt.Sprintf("postgres://%v:%v@%v:%v/%v",
				config.DefaultConfig.DBUser, config.DefaultConfig.DBPass, config.DefaultConfig.DBHost, config.DefaultConfig.DBPort, config.DefaultConfig.DBName)
		}
	case "sqlite3":
		if config.DefaultConfig.DBURL != "" {
			connString = config.DefaultConfig.DBURL
		} else {
			connString = "file::memory:?cache=shared"
		}
	default:
		log.Fatalf("error: unsupported database: %v", config.DefaultConfig.DBDriver)
	}

	db, err := Open(config.DefaultConfig.DBDriver, connString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if config.DefaultConfig.Migrate {
		log.Debug("running migrations")
		if err := db.Migrate(config.DefaultConfig.Reset); err != nil {
			log.Fatal(err)
		}
		log.Debug("migrations complete")

		if config.DefaultConfig.SeedPath != "" {
			log.Debug("seeding database")
			if err := db.Seed(config.DefaultConfig.SeedPath); err != nil {
				log.Fatal(err)
			}
			log.Debug("seed complete")
		}
		os.Exit(0)
	}

	apiroots := strings.Split(config.DefaultConfig.PathPrefix, ",")
	for i, root := range apiroots {
		apiroots[i] = path.Join(root, config.DefaultConfig.AppName, config.DefaultConfig.APIVersion)
	}

	var events *chan []byte
	if config.DefaultConfig.KafkaBootstrap != "" {
		c := make(chan []byte, config.DefaultConfig.EventBuffer)
		events = &c
		ProduceMessages(config.DefaultConfig.KafkaBootstrap, config.DefaultConfig.MetricsTopic, true, events)
		log.WithFields(log.Fields{
			"broker": config.DefaultConfig.KafkaBootstrap,
			"topic":  config.DefaultConfig.MetricsTopic,
		}).Info("started kafka producer")
	}

	srv, err := NewServer(config.DefaultConfig.Addr, apiroots, db, events)
	if err != nil {
		log.Fatal(err)
	}
	defer srv.Close()

	go func() {
		log.WithFields(log.Fields{
			"routine": "db_trim",
		}).Info("started database trimmer")
		for {
			rows, err := db.DeleteEvents(time.Now().UTC().Add(-30 * 24 * time.Hour))
			if err != nil {
				log.WithFields(log.Fields{
					"routine": "db_trim",
					"error":   err,
				}).Error("deleting events")
			}
			log.WithFields(log.Fields{
				"routine": "db_trim",
				"rows":    rows,
			}).Info("deleted rows")
			time.Sleep(1 * time.Hour)
		}
	}()

	go func() {
		log.WithFields(log.Fields{
			"routine": "metrics",
			"addr":    config.DefaultConfig.MAddr,
		}).Info("started http listener")
		if err := http.ListenAndServe(config.DefaultConfig.MAddr, promhttp.Handler()); err != nil {
			log.Fatalf("error: failed to listen to addr (%v): %v", config.DefaultConfig.MAddr, err)
		}
	}()

	go func() {
		log.WithFields(log.Fields{
			"routine": "app",
			"addr":    config.DefaultConfig.Addr,
		}).Info("started http listener")
		log.Fatal(srv.ListenAndServe())
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit
}
