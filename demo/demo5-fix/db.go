package main

import (
	"database/sql"
	"errors"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type appointment struct {
	ID              int64  `json:"id,omitempty"`
	StudentID       string `json:"student_id"`
	AdvisorID       string `json:"advisor_id"`
	AppointmentType string `json:"appointment_type"`
	StartsAt        string `json:"starts_at"`
}

type database struct {
	db *sql.DB
}

type idempotencyRecord struct {
	RequestHash  string
	ResponseBody []byte
	StatusCode   int
	ResourceID   string
}

func newDatabase(path string) (*database, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// Keep the demo deterministic: serialize database operations so the
	// application-level slot race is visible instead of producing SQLITE_BUSY.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS appointments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			student_id TEXT NOT NULL,
			advisor_id TEXT NOT NULL,
			appointment_type TEXT NOT NULL,
			starts_at TEXT NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS idempotency_keys (
			idempotency_key TEXT PRIMARY KEY,
			request_hash TEXT NOT NULL,
			response_body TEXT NOT NULL,
			status_code INTEGER NOT NULL,
			resource_id TEXT NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	// Fixes the race condition by enforcing uniqueness at the database level
	_, err = db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS appointments_advisor_slot
		ON appointments (advisor_id, starts_at)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &database{db: db}, nil
}

func (db *database) close() error {
	return db.db.Close()
}

func (db *database) save(appointment appointment) (int64, error) {
	result, err := db.db.Exec(`
		INSERT INTO appointments (student_id, advisor_id, appointment_type, starts_at)
		VALUES (?, ?, ?, ?)
	`, appointment.StudentID, appointment.AdvisorID, appointment.AppointmentType, appointment.StartsAt)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	return id, err
}

func (db *database) slotTaken(advisorID, startsAt string) (bool, error) {
	var count int
	err := db.db.QueryRow(`
		SELECT COUNT(*)
		FROM appointments
		WHERE advisor_id = ? AND starts_at = ?
	`, advisorID, startsAt).Scan(&count)
	return count > 0, err
}

func isUniqueConstraint(err error) bool {
	var sqliteError *sqlite.Error
	if !errors.As(err, &sqliteError) {
		return false
	}
	return sqliteError.Code()&0xff == sqlite3.SQLITE_CONSTRAINT
}

func (db *database) findIdempotency(key string) (*idempotencyRecord, error) {
	record := &idempotencyRecord{}
	err := db.db.QueryRow(`
		SELECT request_hash, response_body, status_code, resource_id
		FROM idempotency_keys
		WHERE idempotency_key = ?
	`, key).Scan(&record.RequestHash, &record.ResponseBody, &record.StatusCode, &record.ResourceID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (db *database) saveIdempotency(key, requestHash string, responseBody []byte, statusCode int, resourceID string) error {
	_, err := db.db.Exec(`
		INSERT INTO idempotency_keys (idempotency_key, request_hash, response_body, status_code, resource_id)
		VALUES (?, ?, ?, ?, ?)
	`, key, requestHash, responseBody, statusCode, resourceID)
	return err
}
