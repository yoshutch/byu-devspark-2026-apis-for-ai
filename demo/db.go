package main

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

type appointment struct {
	BYUID string `json:"byuId"`
	Type  string `json:"type"`
	Time  string `json:"time"`
	Date  string `json:"date"`
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
			byu_id TEXT NOT NULL,
			type TEXT NOT NULL,
			time TEXT NOT NULL,
			date TEXT NOT NULL
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
		INSERT INTO appointments (byu_id, type, time, date)
		VALUES (?, ?, ?, ?)
	`, appointment.BYUID, appointment.Type, appointment.Time, appointment.Date)
	return err
}
