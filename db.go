package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	_ "github.com/jackc/pgx/v4/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

// DB wraps a sql.DB handle, providing an application-specific, higher-level API
// around the standard sql.DB interface.
type DB struct {
	handle     *sqlx.DB
	statements map[string]*sqlx.Stmt
	driverName string
}

// Open opens a database specified by dataSourceName. The only supported driver
// types are "sqlite3" or "pgx".
//
// Open adheres to all database/sql driver expectations. For example, it is an
// error to request a dataSourceName of ":memory:" with the "sqlite3" driver.
func Open(driverName, dataSourceName string) (*DB, error) {
	handle, err := sqlx.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	if err := handle.Ping(); err != nil {
		return nil, err
	}

	return &DB{
		handle:     handle,
		statements: make(map[string]*sqlx.Stmt),
		driverName: driverName,
	}, nil
}

// Close closes all open prepared statements and returns the connection to the
// connection pool.
func (db *DB) Close() error {
	for _, stmt := range db.statements {
		stmt.Close()
	}
	return db.handle.Close()
}

// Count returns the number of records found in the accounts_modules table with
// the given module name and account ID.
func (db *DB) Count(moduleName, accountID string) (int, error) {
	stmt, err := db.preparedStatement(`SELECT COUNT(*) FROM accounts_modules WHERE module_name = $1 AND account_id = $2;`)
	if err != nil {
		return -1, err
	}

	var count int
	err = stmt.QueryRow(moduleName, accountID).Scan(&count)
	if err != nil {
		return -1, err
	}
	return count, nil
}

// InsertAccountsModules creates a new record in the accounts_modules table with
// the given module name and account ID, creating their respective table records
// if necessary.
func (db *DB) InsertAccountsModules(moduleName, accountID string) error {
	stmt, err := db.preparedStatement(`INSERT INTO accounts_modules (module_name, account_id) VALUES ($1, $2);`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(moduleName, accountID)
	if err != nil {
		return err
	}

	return nil
}

// InsertEvents creates a new record in the events table.
func (db *DB) InsertEvents(phase string, startedAt time.Time, exit int, exception sql.NullString, endedAt time.Time, machineID string, coreVersion string, corePath string) error {
	eventID, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	stmt, err := db.preparedStatement(`INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version, core_path) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(eventID.String(), phase, startedAt, exit, exception, endedAt, machineID, coreVersion, corePath)
	if err != nil {
		return err
	}

	return nil
}

// GetEvents returns a slice of maps loaded with records from the events table.
func (db *DB) GetEvents(limit int, offset int) ([]map[string]interface{}, error) {
	type event struct {
		EventID     string         `db:"event_id"`
		Phase       string         `db:"phase"`
		StartedAt   time.Time      `db:"started_at"`
		Exit        int            `db:"exit"`
		Exception   sql.NullString `db:"exception"`
		EndedAt     time.Time      `db:"ended_at"`
		MachineID   string         `db:"machine_id"`
		CoreVersion string         `db:"core_version"`
		CorePath    sql.NullString `db:"core_path"`
	}
	var stmt *sqlx.Stmt
	if limit < 0 {
		var err error
		stmt, err = db.preparedStatement(`SELECT * FROM events ORDER BY started_at;`)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		stmt, err = db.preparedStatement(fmt.Sprintf(`SELECT * FROM events ORDER BY started_at LIMIT %v OFFSET %v;`, limit, offset))
		if err != nil {
			return nil, err
		}
	}

	rows, err := stmt.Queryx()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]map[string]interface{}, 0)
	for rows.Next() {
		var e event
		if err := rows.StructScan(&e); err != nil {
			return nil, err
		}
		event := make(map[string]interface{})
		event["event_id"] = e.EventID
		event["phase"] = e.Phase
		event["started_at"] = e.StartedAt
		event["exit"] = e.Exit
		if e.Exception.Valid {
			event["exception"] = e.Exception.String
		}
		event["ended_at"] = e.EndedAt
		event["machine_id"] = e.MachineID
		event["core_version"] = e.CoreVersion
		if e.CorePath.Valid {
			event["core_path"] = e.CorePath.String
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

// DeleteEvents deletes all rows from the events table that have a started_at
// date older than the given time and returns the number of rows deleted.
func (db *DB) DeleteEvents(older time.Time) (int64, error) {
	stmt, err := db.preparedStatement(`DELETE FROM events WHERE started_at < $1;`)

	result, err := stmt.Exec(older.Format(time.RFC3339))
	if err != nil {
		return -1, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return -1, err
	}

	return rowsAffected, nil
}

// Migrate inspects the current active migration version and runs all necessary
// steps to migrate all the way up. If reset is true, everything is deleted in
// the database before applying migrations.
func (db *DB) Migrate(reset bool) error {
	m, err := newMigrate(db.handle.DB, db.driverName)
	if err != nil {
		return err
	}

	if reset {
		if err := m.Drop(); err != nil {
			return err
		}
		// After calling Drop, we need to ensure the schema_migrations table
		// exists. In the postgres driver, an unexported function, ensureVersionTable,
		// is called inside WithInstance. So we just reinitialize m to a new
		// Migrate instance.
		m, err = newMigrate(db.handle.DB, db.driverName)
		if err != nil {
			return err
		}
	}

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			return nil
		}
		return err
	}
	return nil
}

// Seed executes the SQL contained in path in order to seed the database.
func (db *DB) Seed(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return db.seedData(data)
}

func (db *DB) seedData(data []byte) error {
	_, err := db.handle.Exec(string(data))
	if err != nil {
		return err
	}
	return nil
}

// preparedStatement creates a prepared statement for the given query, caches
// it in a map and returns the prepared statement. If a statement already exists
// for query, the cached statement is returned.
func (db *DB) preparedStatement(query string) (*sqlx.Stmt, error) {
	stmt := db.statements[query]
	if stmt != nil {
		return stmt, nil
	}
	stmt, err := db.handle.Preparex(query)
	if err != nil {
		return nil, err
	}
	db.statements[query] = stmt
	return stmt, nil
}

func newMigrate(db *sql.DB, driverName string) (*migrate.Migrate, error) {
	var driver database.Driver
	var err error
	switch driverName {
	case "pgx":
		driver, err = postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			return nil, err
		}
	case "sqlite3":
		driver, err = sqlite3.WithInstance(db, &sqlite3.Config{})
		if err != nil {
			return nil, err
		}
	}

	m, err := migrate.NewWithDatabaseInstance("file://./migrations", driverName, driver)
	if err != nil {
		return nil, err
	}

	return m, nil
}
