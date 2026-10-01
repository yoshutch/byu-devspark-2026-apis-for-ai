package main

import (
	"encoding/json"
	"net/http"
)

type problemDocumentation struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Status      int    `json:"status"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Resolution  string `json:"resolution"`
}

func registerProblemRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /problems/authentication-required", problemDocumentationHandler(problemDocumentation{
		Type:        "/problems/authentication-required",
		Title:       "Authentication required",
		Status:      http.StatusUnauthorized,
		Code:        "AUTHENTICATION_REQUIRED",
		Description: "The request did not include a valid bearer token.",
		Resolution:  "Send the request with a valid Authorization: Bearer token header.",
	}))
	mux.HandleFunc("GET /problems/appointment-not-permitted", problemDocumentationHandler(problemDocumentation{
		Type:        "/problems/appointment-not-permitted",
		Title:       "Appointment not permitted",
		Status:      http.StatusForbidden,
		Code:        "APPOINTMENT_NOT_PERMITTED",
		Description: "The caller is authenticated but is not allowed to schedule this appointment.",
		Resolution:  "Use an identity with scheduling permission and schedule only for the permitted student.",
	}))
}

func problemDocumentationHandler(documentation problemDocumentation) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		documentation.Type = problemTypeURL(r, documentation.Type)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(documentation)
	}
}

func problemTypeURL(r *http.Request, path string) string {
	host := r.Host
	if host == "" {
		host = "localhost:8080"
	}
	return "http://" + host + path
}
