package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"test/internal/store"
)

type linkRequest struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

func main() {
	port := getenv("PORT", "8082")
	svc := store.NewMemoryStore()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/links", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req linkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if req.Code == "" || req.URL == "" {
			http.Error(w, "code and url required", http.StatusBadRequest)
			return
		}
		svc.Save(store.Link{Code: req.Code, URL: req.URL})
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("/links/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		code := strings.TrimPrefix(r.URL.Path, "/links/")
		if code == "" {
			http.Error(w, "code required", http.StatusBadRequest)
			return
		}
		link, err := svc.Get(code)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, "failed to fetch", http.StatusInternalServerError)
			return
		}
		writeJSON(w, link)
	})

	log.Printf("storage listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
