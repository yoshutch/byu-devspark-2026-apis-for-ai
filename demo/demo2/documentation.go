package main

import (
	"encoding/json"
	"net/http"
)

type problemDetails struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

type problemDocumentation struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Status      int    `json:"status"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Resolution  string `json:"resolution"`
}

func registerProblemRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /problems/invalid-json", problemDocumentationHandler(problemDocumentation{
		Type:        "/problems/invalid-json",
		Title:       "Invalid JSON request body",
		Status:      http.StatusBadRequest,
		Code:        "INVALID_JSON",
		Description: "The request body is not valid JSON or contains an unsupported field.",
		Resolution:  "Send one JSON object using only the fields supported by the API.",
	}))
	mux.HandleFunc("GET /problems/invalid-appointment", problemDocumentationHandler(problemDocumentation{
		Type:        "/problems/invalid-appointment",
		Title:       "Invalid appointment",
		Status:      http.StatusBadRequest,
		Code:        "INVALID_APPOINTMENT",
		Description: "One or more appointment fields are missing or invalid.",
		Resolution:  "Provide all required fields using the documented formats.",
	}))
	mux.HandleFunc("GET /problems/multiple-json-values", problemDocumentationHandler(problemDocumentation{
		Type:        "/problems/multiple-json-values",
		Title:       "Multiple JSON values",
		Status:      http.StatusBadRequest,
		Code:        "MULTIPLE_JSON_VALUES",
		Description: "The request body contains more than one JSON value.",
		Resolution:  "Send exactly one JSON object in the request body.",
	}))
	mux.HandleFunc("GET /problems/internal-server-error", problemDocumentationHandler(problemDocumentation{
		Type:        "/problems/internal-server-error",
		Title:       "Internal server error",
		Status:      http.StatusInternalServerError,
		Code:        "INTERNAL_SERVER_ERROR",
		Description: "The server could not complete the request.",
		Resolution:  "Retry the request when appropriate, using the same idempotency key if one was provided.",
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

func writeProblem(w http.ResponseWriter, status int, problem problemDetails) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem)
}
