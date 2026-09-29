package main

import "sync"

type appointment struct {
	BYUID string `json:"byuId"`
	Type  string `json:"type"`
	Time  string `json:"time"`
	Date  string `json:"date"`
}

// database keeps the storage details out of the HTTP handler. It can be
// replaced with a file-backed database when a later demo needs persistence.
type database struct {
	mu           sync.Mutex
	appointments []appointment
}

func newDatabase() *database {
	return &database{}
}

func (db *database) save(appointment appointment) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.appointments = append(db.appointments, appointment)
	return nil
}
