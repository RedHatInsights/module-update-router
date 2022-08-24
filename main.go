package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redhatinsights/module-update-router/internal/config"
	log "github.com/sirupsen/logrus"
)

func main() {
	var db *DB

	root := ffcli.Command{
		FlagSet: config.FlagSet(filepath.Base(os.Args[0]), flag.ExitOnError),
		Options: []ff.Option{
			ff.WithEnvVarNoPrefix(),
		},
		Subcommands: []*ffcli.Command{
			{
				Name:      "migrate",
				ShortHelp: "run database migration",
				FlagSet: func() *flag.FlagSet {
					fs := flag.NewFlagSet("migrate", flag.ExitOnError)

					fs.StringVar(&config.DefaultConfig.SeedPath, "seed-path", config.DefaultConfig.SeedPath, "path to the SQL seed file")
					fs.BoolVar(&config.DefaultConfig.Reset, "reset", config.DefaultConfig.Reset, "drop all tables before running migrations")

					return fs
				}(),
				Options: []ff.Option{
					ff.WithEnvVarNoPrefix(),
				},
				Exec: func(ctx context.Context, args []string) error {
					log.Debug("running migrations")
					if err := db.Migrate(config.DefaultConfig.Reset); err != nil {
						return err
					}
					log.Debug("migrations complete")

					if config.DefaultConfig.SeedPath != "" {
						log.Debug("seeding database")
						if err := db.Seed(config.DefaultConfig.SeedPath); err != nil {
							return err
						}
						log.Debug("seed complete")
					}
					return nil
				},
			},
			{
				Name:      "http-api",
				ShortHelp: "run HTTP services",
				Options: []ff.Option{
					ff.WithEnvVarNoPrefix(),
				},
				FlagSet: func() *flag.FlagSet {
					fs := flag.NewFlagSet("http-api", flag.ExitOnError)

					fs.StringVar(&config.DefaultConfig.Addr, "addr", config.DefaultConfig.Addr, "app listen address")
					fs.StringVar(&config.DefaultConfig.APIVersion, "api-version", config.DefaultConfig.APIVersion, "version to use in the URL path")
					fs.StringVar(&config.DefaultConfig.AppName, "app-name", config.DefaultConfig.AppName, "name component for the API prefix")
					fs.IntVar(&config.DefaultConfig.EventBuffer, "event-buffer", config.DefaultConfig.EventBuffer, "the size of the event channel buffer")
					fs.StringVar(&config.DefaultConfig.KafkaBootstrap, "kafka-bootstrap", config.DefaultConfig.KafkaBootstrap, "url of the kafka broker for the cluster")
					fs.StringVar(&config.DefaultConfig.MAddr, "maddr", config.DefaultConfig.MAddr, "metrics listen address")
					fs.StringVar(&config.DefaultConfig.MetricsTopic, "metrics-topic", config.DefaultConfig.MetricsTopic, "topic on which to place metrics data")
					fs.StringVar(&config.DefaultConfig.PathPrefix, "path-prefix", config.DefaultConfig.PathPrefix, "API path prefix")

					return fs
				}(),
				Exec: func(ctx context.Context, args []string) error {
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

					return nil
				},
			},
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}

	if err := root.Parse(os.Args[1:]); err != nil {
		log.Fatalf("error: failed to parse flags: %v", err)
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

	db, err = Open(config.DefaultConfig.DBDriver, connString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := root.Run(context.Background()); err != nil {
		log.Fatalf("error: cannot execute command: %v", err)
	}
}
