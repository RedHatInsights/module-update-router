package main

import (
	"encoding/json"
	"net/http"
	"path"
	"time"

	"github.com/redhatinsights/platform-go-middlewares/identity"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	metrics "github.com/slok/go-http-metrics/metrics/prometheus"
	"github.com/slok/go-http-metrics/middleware"

	request "github.com/redhatinsights/platform-go-middlewares/request_id"
)

// Server is the application's HTTP server. It is comprised of an HTTP
// multiplexer for routing HTTP requests to appropriate handlers and a database
// handle for looking up application data.
type Server struct {
	mux  *http.ServeMux
	db   *DB
	addr string
}

// NewServer creates a new instance of the application, configured with the
// provided addr, dbpath, and API root.
func NewServer(addr, dbpath, apiroot string) (*Server, error) {
	db, err := Open(dbpath)
	if err != nil {
		return nil, err
	}
	srv := &Server{
		mux:  &http.ServeMux{},
		db:   db,
		addr: addr,
	}
	srv.routes(apiroot)
	return srv, nil
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// ListenAndServe simply calls http.ListenAndServe with the configured TCP
// address and s as the handler.
func (s Server) ListenAndServe() error {
	return http.ListenAndServe(s.addr, s)
}

// Close closes the database handle.
func (s *Server) Close() error {
	return s.db.Close()
}

// routes registers handlerFuncs for the server paths under the given prefix.
func (s *Server) routes(prefix string) {
	s.mux.HandleFunc("/ping", s.handlePing())
	s.mux.HandleFunc(prefix+"/", s.metrics(s.requestID(s.log(s.auth(s.handleAPI(prefix))))))
}

// handlePing creates an http.HandlerFunc that handles the health check endpoint
// /ping.
func (s *Server) handlePing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`OK`))
	}
}

// handleAPI creates an http.HandlerFunc that creates handlerFuncs for
// operations under the API root.
func (s *Server) handleAPI(prefix string) http.HandlerFunc {
	m := http.ServeMux{}

	m.HandleFunc(path.Join(prefix, "channel"), s.handleChannel())

	return func(w http.ResponseWriter, r *http.Request) {
		m.ServeHTTP(w, r)
	}
}

// handleChannel creates an http.HandlerFunc for the API endpoint /channel.
func (s *Server) handleChannel() http.HandlerFunc {
	type response struct {
		URL string `json:"url"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		module := r.URL.Query().Get("module")
		if len(module) < 1 {
			formatJSONError(w, http.StatusBadRequest, "missing required paramenter: 'module'")
			return
		}

		resp := response{
			URL: "/release",
		}
		id := identity.Get(r.Context())
		count, err := s.db.Count(module, id.Identity.AccountNumber)
		if err != nil {
			log.Error(err)
		}
		if count > 0 {
			resp.URL = "/testing"
		}
		data, err := json.Marshal(resp)
		if err != nil {
			formatJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Write(data)
	}
}

// log is an http HandlerFunc middlware handler that creates a responseWriter
// and logs details about the HandlerFunc it wraps.
func (s *Server) log(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rr := newResponseRecorder(w)
		start := time.Now()

		next(rr, r)

		var level log.Level
		switch {
		case rr.Code >= 400:
			level = log.WarnLevel
		case rr.Code >= 500:
			level = log.ErrorLevel
		default:
			level = log.InfoLevel
		}

		log.WithFields(logrus.Fields{
			"ident":          r.Host,
			"method":         r.Method,
			"referer":        r.Referer(),
			"url":            r.URL.String(),
			"user-agent":     r.UserAgent(),
			"status":         rr.Code,
			"response":       rr.Body.String(),
			"account-number": identity.Get(r.Context()).Identity.AccountNumber,
			"duration":       time.Since(start),
		}).Log(level)
	}
}

// requestID is an http HandlerFunc middleware handler that creates a request ID
// and writes it to the response header map.
func (s *Server) requestID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request.RequestID(next).ServeHTTP(w, r)
	}
}

// auth is an http HandlerFunc middleware handler that ensures a valid
// X-Rh-Identity header is present in the request.
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity.EnforceIdentity(next).ServeHTTP(w, r)
	}
}

// metrics is an http HandlerFunc middleware handler that creates and enables
// a metrics recorder.
func (s *Server) metrics(next http.HandlerFunc) http.HandlerFunc {
	m := middleware.New(middleware.Config{
		Recorder: metrics.NewRecorder(metrics.Config{}),
	})
	return func(w http.ResponseWriter, r *http.Request) {
		m.Handler("", http.Handler(next)).ServeHTTP(w, r)
	}
}
