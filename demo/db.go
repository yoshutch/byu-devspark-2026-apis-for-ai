package main

import (
	"database/sql"
	"errors"

	_ "modernc.org/sqlite"
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
