package main

import (
	"log/slog"
	"net/http"
	"os"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	databasePath := os.Getenv("DB_PATH")
	if databasePath == "" {
		databasePath = "appointments.db"
	}

	db, err := newDatabase(databasePath)
	if err != nil {
		slog.Error("could not open database", "error", err)
		return
	}
	defer db.close()

	mux := http.NewServeMux()
	registerProblemRoutes(mux)
	mux.HandleFunc("POST /advisement/appointments", createAppointment(db))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("demo API listening", "url", "http://localhost:"+port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		slog.Error("server stopped", "error", err)
	}
}
