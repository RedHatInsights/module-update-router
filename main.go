package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"

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
		Exec: func(ctx context.Context, args []string) error {
			switch config.DefaultConfig.LogFormat.Value {
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

			db, err = Open("sqlite", "file::memory:?cache=shared")
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()

			log.Debug("running migrations")
			if err := db.Migrate(config.DefaultConfig.Reset); err != nil {
				return err
			}
			log.Debug("migrations complete")

			if config.DefaultConfig.SeedPath.Value != "" {
				log.Debug("seeding database")
				if err := db.Seed(config.DefaultConfig.SeedPath.Value); err != nil {
					return err
				}
				log.Debug("seed complete")
			}

			apiroots := strings.Split(config.DefaultConfig.PathPrefix, ",")
			for i, root := range apiroots {
				apiroots[i] = path.Join(root, config.DefaultConfig.AppName, config.DefaultConfig.APIVersion)
			}

			srv, err := NewServer(config.DefaultConfig.Addr, apiroots, db)
			if err != nil {
				log.Fatal(err)
			}
			defer srv.Close()

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
	}

	if err := root.Parse(os.Args[1:]); err != nil {
		log.Fatalf("error: failed to parse flags: %v", err)
	}

	if err := root.Run(context.Background()); err != nil {
		log.Fatalf("error: cannot execute command: %v", err)
	}
}
