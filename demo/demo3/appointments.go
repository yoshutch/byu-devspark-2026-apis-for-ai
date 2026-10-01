package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type principal struct {
	ActorID          string
	ClientID         string
	CanCreate        bool
	AllowedStudentID string
}

var demoPrincipals = map[string]principal{
	"demo-readonly": {
		ActorID:  "user-999999999",
		ClientID: "demo-cli",
	},
	"demo-student": {
		ActorID:          "user-123456789",
		ClientID:         "demo-cli",
		CanCreate:        true,
		AllowedStudentID: "123456789",
	},
}

type problemDetails struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func createAppointment(db *database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := getRequestID(r)
		caller, authenticated := authenticate(r)
		if !authenticated {
			auditAppointmentAttempt(r, principal{}, requestID, "rejected", "authentication_required", http.StatusUnauthorized, "")
			writeProblem(w, http.StatusUnauthorized, problemDetails{
				Type:   problemTypeURL(r, "/problems/authentication-required"),
				Title:  "Authentication required",
				Status: http.StatusUnauthorized,
				Code:   "AUTHENTICATION_REQUIRED",
				Detail: "Provide a valid bearer token.",
			})
			return
		}

		if !caller.CanCreate {
			auditAppointmentAttempt(r, caller, requestID, "rejected", "insufficient_permission", http.StatusForbidden, "")
			writeProblem(w, http.StatusForbidden, problemDetails{
				Type:   problemTypeURL(r, "/problems/appointment-not-permitted"),
				Title:  "Appointment not permitted",
				Status: http.StatusForbidden,
				Code:   "APPOINTMENT_NOT_PERMITTED",
				Detail: "The caller is not allowed to schedule this appointment.",
			})
			return
		}

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		var newAppointment appointment
		if err := decoder.Decode(&newAppointment); err != nil {
			auditAppointmentAttempt(r, caller, requestID, "rejected", "invalid_json", http.StatusBadRequest, "")
			http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
			return
		}

		if err := validateAppointment(newAppointment); err != nil {
			auditAppointmentAttempt(r, caller, requestID, "rejected", "invalid_request", http.StatusBadRequest, "")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var extra json.RawMessage
		if err := decoder.Decode(&extra); err != io.EOF {
			auditAppointmentAttempt(r, caller, requestID, "rejected", "multiple_json_values", http.StatusBadRequest, "")
			http.Error(w, "request body must contain one JSON object", http.StatusBadRequest)
			return
		}

		if caller.AllowedStudentID != "" && caller.AllowedStudentID != newAppointment.StudentID {
			auditAppointmentAttempt(r, caller, requestID, "rejected", "appointment_for_another_student", http.StatusForbidden, "")
			writeProblem(w, http.StatusForbidden, problemDetails{
				Type:   problemTypeURL(r, "/problems/appointment-not-permitted"),
				Title:  "Appointment not permitted",
				Status: http.StatusForbidden,
				Code:   "APPOINTMENT_NOT_PERMITTED",
				Detail: "The caller can only schedule appointments for themselves.",
			})
			return
		}

		id, err := db.save(newAppointment)
		if err != nil {
			auditAppointmentAttempt(r, caller, requestID, "failed", "database_error", http.StatusInternalServerError, "")
			http.Error(w, "could not save appointment", http.StatusInternalServerError)
			return
		}
		newAppointment.ID = id
		resourceID := fmt.Sprintf("appointment-%d", id)
		auditAppointmentAttempt(r, caller, requestID, "success", "", http.StatusOK, resourceID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newAppointment)
	}
}

func auditAppointmentAttempt(r *http.Request, caller principal, requestID, outcome, reason string, status int, resourceID string) {
	slog.Info("audit",
		"event", "appointment.create",
		"request_id", requestID,
		"actor_id", nullableString(caller.ActorID),
		"client_id", nullableString(caller.ClientID),
		"resource_id", resourceID,
		"outcome", outcome,
		"reason", reason,
		"status", status,
		"method", r.Method,
		"path", r.URL.Path,
	)
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func authenticate(r *http.Request) (principal, bool) {
	const bearerPrefix = "Bearer "
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authorization, bearerPrefix) {
		return principal{}, false
	}

	token := strings.TrimSpace(strings.TrimPrefix(authorization, bearerPrefix))
	caller, ok := demoPrincipals[token]
	return caller, ok
}

func getRequestID(r *http.Request) string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

func writeProblem(w http.ResponseWriter, status int, problem problemDetails) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(problem)
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
