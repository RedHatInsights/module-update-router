package main

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	_ "modernc.org/sqlite"
)

//go:embed migrations
var migrations embed.FS

// DB wraps a sql.DB handle, providing an application-specific, higher-level API
// around the standard sql.DB interface.
type DB struct {
	handle     *sqlx.DB
	statements map[string]*sqlx.Stmt
	driverName string
}

// Open opens a database specified by dataSourceName. The only supported driver
// types are "sqlite".
func Open(driverName, dataSourceName string) (*DB, error) {
	handle, err := sqlx.Open(driverName, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("db: sqlx.Open failed: %w", err)
	}

	if err := handle.Ping(); err != nil {
		return nil, fmt.Errorf("db: handle.Ping failed: %w", err)
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

// Count returns the number of records found in the orgs_modules table with the
// given module name and org ID.
func (db *DB) Count(moduleName, orgID string) (int, error) {
	stmt, err := db.preparedStatement(`SELECT COUNT(*) FROM orgs_modules WHERE module_name = $1 AND org_id = $2;`)
	if err != nil {
		return -1, fmt.Errorf("db: db.preparedStatement failed: %w", err)
	}

	var count int
	err = stmt.QueryRow(moduleName, orgID).Scan(&count)
	if err != nil {
		return -1, fmt.Errorf("db: stmt.QueryRow failed: %w", err)
	}
	return count, nil
}

// InsertOrgsModules creates a new record in the orgs_modules table with the
// given module name and org ID, creating their respective table records if
// necessary.
func (db *DB) InsertOrgsModules(moduleName, orgID string) error {
	stmt, err := db.preparedStatement(`INSERT INTO orgs_modules (module_name, org_id) VALUES ($1, $2);`)
	if err != nil {
		return fmt.Errorf("db: db.preparedStatement failed: %w", err)
	}
	_, err = stmt.Exec(moduleName, orgID)
	if err != nil {
		return fmt.Errorf("db: stmt.Exec failed: %w", err)
	}

	return nil
}

// InsertEvents creates a new record in the events table.
func (db *DB) InsertEvents(phase string, startedAt time.Time, exit int, exception sql.NullString, endedAt time.Time, machineID string, coreVersion string, corePath string) error {
	eventID, err := uuid.NewUUID()
	if err != nil {
		return fmt.Errorf("db: uuid.NewUUID failed: %w", err)
	}

	stmt, err := db.preparedStatement(`INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version, core_path) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);`)
	if err != nil {
		return fmt.Errorf("db: db.preparedStatement failed: %w", err)
	}

	_, err = stmt.Exec(eventID.String(), phase, startedAt, exit, exception, endedAt, machineID, coreVersion, corePath)
	if err != nil {
		return fmt.Errorf("db: stmt.Exec failed: %w", err)
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
			return nil, fmt.Errorf("db: db.preparedStatement failed: %w", err)
		}
	} else {
		var err error
		stmt, err = db.preparedStatement(fmt.Sprintf(`SELECT * FROM events ORDER BY started_at LIMIT %v OFFSET %v;`, limit, offset))
		if err != nil {
			return nil, fmt.Errorf("db: db.preparedStatement failed: %w", err)
		}
	}

	rows, err := stmt.Queryx()
	if err != nil {
		return nil, fmt.Errorf("db: stmt.Queryx failed: %w", err)
	}
	defer rows.Close()

	events := make([]map[string]interface{}, 0)
	for rows.Next() {
		var e event
		if err := rows.StructScan(&e); err != nil {
			return nil, fmt.Errorf("db: rows.StructScan failed: %w", err)
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
		return nil, fmt.Errorf("db: rows.Err failed: %w", err)
	}
	return events, nil
}

// DeleteEvents deletes all rows from the events table that have a started_at
// date older than the given time and returns the number of rows deleted.
func (db *DB) DeleteEvents(older time.Time) (int64, error) {
	stmt, err := db.preparedStatement(`DELETE FROM events WHERE started_at < $1;`)
	if err != nil {
		return -1, fmt.Errorf("db: db.preparedStatement failed: %w", err)
	}

	result, err := stmt.Exec(older.Format(time.RFC3339))
	if err != nil {
		return -1, fmt.Errorf("db: stmt.Exec failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return -1, fmt.Errorf("db: result.RowsAffected failed: %w", err)
	}

	return rowsAffected, nil
}

// Migrate inspects the current active migration version and runs all necessary
// steps to migrate all the way up. If reset is true, everything is deleted in
// the database before applying migrations.
func (db *DB) Migrate(reset bool) error {
	m, err := newMigrate(db.handle.DB, db.driverName)
	if err != nil {
		return fmt.Errorf("db: newMigrate failed: %w", err)
	}

	if reset {
		if err := m.Drop(); err != nil {
			return fmt.Errorf("db: m.Drop failed: %w", err)
		}
		// After calling Drop, we need to ensure the schema_migrations table
		// exists. In the postgres driver, an unexported function, ensureVersionTable,
		// is called inside WithInstance. So we just reinitialize m to a new
		// Migrate instance.
		m, err = newMigrate(db.handle.DB, db.driverName)
		if err != nil {
			return fmt.Errorf("db: newMigrate failed: %w", err)
		}
	}

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			return nil
		}
		return fmt.Errorf("db: m.Up failed: %w", err)
	}
	return nil
}

// Seed executes the SQL contained in path in order to seed the database.
func (db *DB) Seed(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("db: os.ReadFile failed: %w", err)
	}
	return db.seedData(data)
}

func (db *DB) seedData(data []byte) error {
	_, err := db.handle.Exec(string(data))
	if err != nil {
		return fmt.Errorf("db: db.handle.Exec failed: %w", err)
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
		return nil, fmt.Errorf("db: db.handle.Preparex failed: %w", err)
	}
	db.statements[query] = stmt
	return stmt, nil
}

func newMigrate(db *sql.DB, driverName string) (*migrate.Migrate, error) {
	driver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return nil, fmt.Errorf("db: sqlite.WithInstance failed: %w", err)
	}

	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithInstance("iofs", source, driverName, driver)
	if err != nil {
		return nil, fmt.Errorf("db: migrate.NewithInstance failed: %w", err)
	}

	return m, nil
}
