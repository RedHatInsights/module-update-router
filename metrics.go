package main

import (
	"time"

	p "github.com/prometheus/client_golang/prometheus"
	pa "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requests = pa.NewCounterVec(p.CounterOpts{
		Name: "module_update_router_requests",
		Help: "Total number of GETs to router",
	}, []string{"endpoint"})

	events = pa.NewCounterVec(p.CounterOpts{
		Name: "module_update_router_events",
		Help: "Total number of events recorded",
	}, []string{"core_version"})

	clientElapsed = pa.NewHistogramVec(p.HistogramOpts{
		Name: "module_update_router_client_seconds",
		Help: "The number of seconds a client",
	}, []string{"phase"})
)

func incRequests(endpoint string) {
	requests.With(p.Labels{"endpoint": endpoint}).Inc()
}

func incEvents(coreVersion string) {
	events.With(p.Labels{"core_version": coreVersion}).Inc()
}

func observeClientElapsed(phase string, startTime time.Time, endTime time.Time) {
	duration := endTime.Sub(startTime)
	clientElapsed.With(p.Labels{"phase": phase}).Observe(duration.Seconds())
}
