package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"

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
	stmt, err := db.preparedStatement(`SELECT COUNT(*) FROM accounts_modules WHERE module_name = ? AND account_id = ?;`)
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

// Insert creates a new record in the accounts_modules table with the given
// module name and account ID, creating their respective table records if
// necessary.
func (db *DB) Insert(moduleName, accountID string) error {
	stmt, err := db.preparedStatement(`INSERT OR IGNORE INTO accounts_modules (module_name, account_id) VALUES (?, ?);`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(moduleName, accountID)
	if err != nil {
		return err
	}

	return nil
}

// Load populates the database with values read from data. It is assumed that
// data is a list of comma-separated values with module name in column 0 and
// account ID in column 1.
//
// Example:
//
//   modfoo,123
//   modfoo,345
//   modboo,123
//   modboo,678
func (db *DB) Load(data string) error {
	r := csv.NewReader(strings.NewReader(data))

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(rec) != 2 {
			return fmt.Errorf("invalid row length: %v", rec)
		}
		if err := db.Insert(rec[0], rec[1]); err != nil {
			return err
		}
	}
	return nil
}

// Migrate inspects the current active migration version and runs all necessary
// steps to migrate all the way up.
func (db *DB) Migrate() error {
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

	if err := m.Up(); err != nil {
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
