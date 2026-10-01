package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func schedule(db *database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		newAppointment := appointment{
			BYUID: "123456789",
			Type:  "advisor",
			Time:  r.PostFormValue("time"),
			Date:  r.PostFormValue("date"),
		}

		if err := db.save(newAppointment); err != nil {
			http.Error(w, "could not save appointment", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newAppointment)
	}
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
	mux.HandleFunc("POST /schedule", schedule(db))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("demo API listening on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
