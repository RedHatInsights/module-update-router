package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"reflect"
	"time"

	"github.com/redhatinsights/platform-go-middlewares/identity"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/slok/go-http-metrics/metrics"
	httpmetrics "github.com/slok/go-http-metrics/metrics/prometheus"
	"github.com/slok/go-http-metrics/middleware"

	request "github.com/redhatinsights/platform-go-middlewares/request_id"
)

var r metrics.Recorder = httpmetrics.NewRecorder(httpmetrics.Config{})

// Server is the application's HTTP server. It is comprised of an HTTP
// multiplexer for routing HTTP requests to appropriate handlers and a database
// handle for looking up application data.
type Server struct {
	mux  *http.ServeMux
	db   *DB
	addr string
}

// NewServer creates a new instance of the application, configured with the
// provided addr, API roots and database handle.
func NewServer(addr string, apiroots []string, db *DB) (*Server, error) {
	srv := &Server{
		mux:  &http.ServeMux{},
		db:   db,
		addr: addr,
	}
	srv.routes(apiroots...)
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

// routes registers handlerFuncs for the server paths under the given prefixes.
func (s *Server) routes(prefixes ...string) {
	s.mux.HandleFunc("/ping", s.handlePing())
	for _, prefix := range prefixes {
		s.mux.HandleFunc(prefix+"/", s.metrics(s.requestID(s.log(s.auth(s.handleAPI(prefix))))))
	}
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
	m.HandleFunc(path.Join(prefix, "event"), s.handleEvent())

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

// handleEvent creates an http.HandlerFunc for the API endpoint /event.
func (s *Server) handleEvent() http.HandlerFunc {
	type requestBody struct {
		Phase       *string    `json:"phase"`
		StartedAt   *time.Time `json:"started_at"`
		Exit        *int       `json:"exit"`
		Exception   *string    `json:"exception"`
		EndedAt     *time.Time `json:"ended_at"`
		MachineID   *string    `json:"machine_id"`
		CoreVersion *string    `json:"core_version"`
		CorePath    *string    `json:"core_path"`
	}
	type event struct {
		Phase       string
		StartedAt   time.Time
		Exit        int
		Exception   sql.NullString
		EndedAt     time.Time
		MachineID   string
		CoreVersion string
		CorePath    string
	}
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			formatJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		defer r.Body.Close()

		var body requestBody
		if err := json.Unmarshal(data, &body); err != nil {
			formatJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Check required fields for presence of value. Each struct member is
		// compared to the zero value of its respective type.
		for p, v := range map[string]interface{}{
			"phase":        body.Phase,
			"started_at":   body.StartedAt,
			"exit":         body.Exit,
			"ended_at":     body.EndedAt,
			"machine_id":   body.MachineID,
			"core_version": body.CoreVersion,
			"core_path":    body.CorePath,
		} {
			if reflect.ValueOf(v) == reflect.Zero(reflect.TypeOf(v)) {
				formatJSONError(w, http.StatusBadRequest, fmt.Sprintf(`missing required field: '%v'`, p))
				return
			}
		}

		e := event{
			Phase:       *body.Phase,
			StartedAt:   *body.StartedAt,
			Exit:        *body.Exit,
			Exception:   NewNullString(body.Exception),
			EndedAt:     *body.EndedAt,
			MachineID:   *body.MachineID,
			CoreVersion: *body.CoreVersion,
			CorePath:    *body.CorePath,
		}

		if err := s.db.InsertEvents(e.Phase, e.StartedAt, e.Exit, e.Exception, e.EndedAt, e.MachineID, e.CoreVersion, e.CorePath); err != nil {
			formatJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusCreated)
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
			"ident":      r.Host,
			"method":     r.Method,
			"referer":    r.Referer(),
			"url":        r.URL.String(),
			"user-agent": r.UserAgent(),
			"status":     rr.Code,
			"response":   rr.Body.String(),
			"duration":   time.Since(start),
			"request-id": r.Header.Get("X-Request-Id"),
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
		Recorder: r,
	})
	return func(w http.ResponseWriter, r *http.Request) {
		m.Handler("", http.Handler(next)).ServeHTTP(w, r)
	}
}
