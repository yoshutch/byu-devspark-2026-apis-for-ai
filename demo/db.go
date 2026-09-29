package main

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

type appointment struct {
	StudentID       string `json:"student_id"`
	AdvisorID       string `json:"advisor_id"`
	AppointmentType string `json:"appointment_type"`
	StartsAt        string `json:"starts_at"`
}

type database struct {
	db *sql.DB
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

	return &database{db: db}, nil
}

func (db *database) close() error {
	return db.db.Close()
}

func (db *database) save(appointment appointment) error {
	_, err := db.db.Exec(`
		INSERT INTO appointments (student_id, advisor_id, appointment_type, starts_at)
		VALUES (?, ?, ?, ?)
	`, appointment.StudentID, appointment.AdvisorID, appointment.AppointmentType, appointment.StartsAt)
	return err
}
