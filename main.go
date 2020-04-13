package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

func main() {
	cfg := DefaultConfig()

	switch cfg.Environment {
	case "production":
		log.SetFormatter(&log.JSONFormatter{})
	default:
		log.SetFormatter(&log.TextFormatter{})
	}

	srv, err := NewServer(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer srv.Close()

	data := DefaultEnv("DB_DATA", "")
	if len(data) > 0 {
		if err := srv.db.Load(data); err != nil {
			log.Fatal(err)
		}
	}

	go func() {
		log.WithFields(log.Fields{
			"func": "metrics",
			"addr": cfg.Maddr,
		}).Info("started")
		http.ListenAndServe(cfg.Maddr, promhttp.Handler())
	}()

	go func() {
		log.WithFields(log.Fields{
			"func": "app",
			"addr": cfg.Addr,
		}).Info("started")
		log.Fatal(srv.ListenAndServe())
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit
}
