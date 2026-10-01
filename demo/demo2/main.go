package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func createAppointment(db *database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		var newAppointment appointment
		if err := decoder.Decode(&newAppointment); err != nil {
			http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
			return
		}

		if err := validateAppointment(newAppointment); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var extra json.RawMessage
		if err := decoder.Decode(&extra); err != io.EOF {
			http.Error(w, "request body must contain one JSON object", http.StatusBadRequest)
			return
		}

		if err := db.save(newAppointment); err != nil {
			http.Error(w, "could not save appointment", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newAppointment)
	}
}

func validateAppointment(newAppointment appointment) error {
	if strings.TrimSpace(newAppointment.StudentID) == "" {
		return fmt.Errorf("student_id is required")
	}
	if strings.TrimSpace(newAppointment.AdvisorID) == "" {
		return fmt.Errorf("advisor_id is required")
	}
	if strings.TrimSpace(newAppointment.AppointmentType) == "" {
		return fmt.Errorf("appointment_type is required")
	}
	if _, err := time.Parse(time.RFC3339, newAppointment.StartsAt); err != nil {
		return fmt.Errorf("starts_at must be a valid RFC3339 timestamp")
	}
	return nil
}

func main() {
	databasePath := os.Getenv("DB_PATH")
	if databasePath == "" {
		databasePath = "appointments.db"
	}

	db, err := newDatabase(databasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.close()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /advisement/appointments", createAppointment(db))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("demo API listening on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
