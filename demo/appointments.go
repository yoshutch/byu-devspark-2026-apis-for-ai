package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
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

type demoConfig struct {
	timeoutAfterCreate time.Duration
	timeoutOnce        sync.Once
}

type problemDetails struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func newDemoConfig() (*demoConfig, error) {
	value := strings.TrimSpace(os.Getenv("DEMO_TIMEOUT_AFTER_CREATE"))
	if value == "" {
		return &demoConfig{}, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return nil, fmt.Errorf("DEMO_TIMEOUT_AFTER_CREATE must be a positive duration such as 3s")
	}
	return &demoConfig{timeoutAfterCreate: duration}, nil
}

func (config *demoConfig) delayAfterCreate() {
	if config.timeoutAfterCreate <= 0 {
		return
	}
	config.timeoutOnce.Do(func() {
		time.Sleep(config.timeoutAfterCreate)
	})
}

func createAppointment(db *database, config *demoConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := getRequestID(r)
		caller, authenticated := authenticate(r)
		if !authenticated {
			auditAppointmentAttempt(r, principal{}, requestID, "rejected", "authentication_required", http.StatusUnauthorized, "")
			writeProblem(w, http.StatusUnauthorized, problemDetails{
				Type:   "https://example.edu/problems/authentication-required",
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
				Type:   "https://example.edu/problems/appointment-not-permitted",
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
				Type:   "https://example.edu/problems/appointment-not-permitted",
				Title:  "Appointment not permitted",
				Status: http.StatusForbidden,
				Code:   "APPOINTMENT_NOT_PERMITTED",
				Detail: "The caller can only schedule appointments for themselves.",
			})
			return
		}

		idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		requestHash := appointmentFingerprint(newAppointment)
		if idempotencyKey != "" {
			record, err := db.findIdempotency(idempotencyKey)
			if err != nil {
				auditAppointmentAttempt(r, caller, requestID, "failed", "idempotency_lookup_error", http.StatusInternalServerError, "")
				http.Error(w, "could not read idempotency record", http.StatusInternalServerError)
				return
			}
			if record != nil {
				if record.RequestHash != requestHash {
					auditAppointmentAttempt(r, caller, requestID, "rejected", "idempotency_key_reused", http.StatusConflict, record.ResourceID)
					http.Error(w, "idempotency key was already used with different request data", http.StatusConflict)
					return
				}

				auditAppointmentAttempt(r, caller, requestID, "replayed", "", record.StatusCode, record.ResourceID)
				writeJSON(w, record.StatusCode, record.ResponseBody)
				return
			}
		}

		id, err := db.save(newAppointment)
		if err != nil {
			auditAppointmentAttempt(r, caller, requestID, "failed", "database_error", http.StatusInternalServerError, "")
			http.Error(w, "could not save appointment", http.StatusInternalServerError)
			return
		}
		newAppointment.ID = id
		resourceID := fmt.Sprintf("appointment-%d", id)
		responseBody, err := json.Marshal(newAppointment)
		if err != nil {
			auditAppointmentAttempt(r, caller, requestID, "failed", "response_encoding_error", http.StatusInternalServerError, resourceID)
			http.Error(w, "could not encode appointment", http.StatusInternalServerError)
			return
		}
		responseBody = append(responseBody, '\n')

		if idempotencyKey != "" {
			if err := db.saveIdempotency(idempotencyKey, requestHash, responseBody, http.StatusOK, resourceID); err != nil {
				auditAppointmentAttempt(r, caller, requestID, "failed", "idempotency_store_error", http.StatusInternalServerError, resourceID)
				http.Error(w, "could not save idempotency record", http.StatusInternalServerError)
				return
			}
		}

		auditAppointmentAttempt(r, caller, requestID, "success", "", http.StatusOK, resourceID)
		config.delayAfterCreate()

		writeJSON(w, http.StatusOK, responseBody)
	}
}

func appointmentFingerprint(newAppointment appointment) string {
	encoded, _ := json.Marshal(newAppointment)
	hash := sha256.Sum256(encoded)
	return hex.EncodeToString(hash[:])
}

func writeJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
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
	if requestID := strings.TrimSpace(r.Header.Get("X-Request-ID")); requestID != "" {
		return requestID
	}
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
