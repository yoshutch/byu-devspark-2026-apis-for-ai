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
			writeProblem(w, http.StatusBadRequest, problemDetails{
				Type:   problemTypeURL(r, "/problems/invalid-json"),
				Title:  "Invalid JSON request body",
				Status: http.StatusBadRequest,
				Code:   "INVALID_JSON",
				Detail: "The request body must contain valid JSON with only supported fields.",
			})
			return
		}

		if err := validateAppointment(newAppointment); err != nil {
			writeProblem(w, http.StatusBadRequest, problemDetails{
				Type:   problemTypeURL(r, "/problems/invalid-appointment"),
				Title:  "Invalid appointment",
				Status: http.StatusBadRequest,
				Code:   "INVALID_APPOINTMENT",
				Detail: err.Error(),
			})
			return
		}

		var extra json.RawMessage
		if err := decoder.Decode(&extra); err != io.EOF {
			writeProblem(w, http.StatusBadRequest, problemDetails{
				Type:   problemTypeURL(r, "/problems/multiple-json-values"),
				Title:  "Multiple JSON values",
				Status: http.StatusBadRequest,
				Code:   "MULTIPLE_JSON_VALUES",
				Detail: "The request body must contain exactly one JSON object.",
			})
			return
		}

		if err := db.save(newAppointment); err != nil {
			writeProblem(w, http.StatusInternalServerError, problemDetails{
				Type:   problemTypeURL(r, "/problems/internal-server-error"),
				Title:  "Internal server error",
				Status: http.StatusInternalServerError,
				Code:   "INTERNAL_SERVER_ERROR",
				Detail: "The server could not complete the request.",
			})
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
	registerProblemRoutes(mux)
	mux.HandleFunc("POST /advisement/appointments", createAppointment(db))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("demo API listening on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
