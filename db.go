package main

import (
	"database/sql"
	"io/ioutil"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"

	_ "github.com/jackc/pgx/v4/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

// DB wraps a sql.DB handle, providing an application-specific, higher-level API
// around the standard sql.DB interface.
type DB struct {
	handle     *sql.DB
	statements map[string]*sql.Stmt
	driverName string
}

// Open opens a database specified by dataSourceName. The only supported driver
// types are "sqlite3" or "pgx".
//
// Open adheres to all database/sql driver expectations. For example, it is an
// error to request a dataSourceName of ":memory:" with the "sqlite3" driver.
func Open(driverName, dataSourceName string) (*DB, error) {
	handle, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	if err := handle.Ping(); err != nil {
		return nil, err
	}

	return &DB{
		handle:     handle,
		statements: make(map[string]*sql.Stmt),
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
func (db *DB) InsertEvents(phase string, startedAt time.Time, exit int, exception sql.NullString, endedAt time.Time, machineID string, coreVersion string) error {
	eventID, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	stmt, err := db.preparedStatement(`INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(eventID.String(), phase, startedAt, exit, exception, endedAt, machineID, coreVersion)
	if err != nil {
		return err
	}

	return nil
}

// Migrate inspects the current active migration version and runs all necessary
// steps to migrate all the way up. If reset is true, everything is deleted in
// the database before applying migrations.
func (db *DB) Migrate(reset bool) error {
	var driver database.Driver
	var err error
	switch db.driverName {
	case "pgx":
		driver, err = postgres.WithInstance(db.handle, &postgres.Config{})
		if err != nil {
			return err
		}
	case "sqlite3":
		driver, err = sqlite3.WithInstance(db.handle, &sqlite3.Config{})
		if err != nil {
			return err
		}
	}

	m, err := migrate.NewWithDatabaseInstance("file://./migrations", db.driverName, driver)
	if err != nil {
		return err
	}

	if reset {
		if err := m.Drop(); err != nil {
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
func (db *DB) preparedStatement(query string) (*sql.Stmt, error) {
	stmt := db.statements[query]
	if stmt != nil {
		return stmt, nil
	}
	stmt, err := db.handle.Prepare(query)
	if err != nil {
		return nil, err
	}
	db.statements[query] = stmt
	return stmt, nil
}
