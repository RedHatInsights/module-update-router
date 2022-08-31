package main

import (
	p "github.com/prometheus/client_golang/prometheus"
	pa "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requests = pa.NewCounterVec(p.CounterOpts{
		Name: "module_update_router_requests",
		Help: "Total number of GETs to router",
	}, []string{"endpoint"})
)

func incRequests(endpoint string) {
	requests.With(p.Labels{"endpoint": endpoint}).Inc()
}
