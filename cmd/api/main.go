package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"test/internal/codec"
)

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

type storagePayload struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

func main() {
	port := getenv("PORT", "8080")
	storageURL := strings.TrimRight(getenv("STORAGE_URL", "http://localhost:8082"), "/")
	baseURL := strings.TrimRight(getenv("BASE_URL", "http://localhost:8081"), "/")

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/shorten", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req shortenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if req.URL == "" {
			http.Error(w, "url required", http.StatusBadRequest)
			return
		}
		if _, err := url.ParseRequestURI(req.URL); err != nil {
			http.Error(w, "invalid url", http.StatusBadRequest)
			return
		}
		code, err := codec.NewCode()
		if err != nil {
			http.Error(w, "failed to generate code", http.StatusInternalServerError)
			return
		}
		payload := storagePayload{Code: code, URL: req.URL}
		body, err := json.Marshal(payload)
		if err != nil {
			http.Error(w, "failed to encode payload", http.StatusInternalServerError)
			return
		}
		resp, err := http.Post(storageURL+"/links", "application/json", bytes.NewReader(body))
		if err != nil {
			http.Error(w, "storage unavailable", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusBadRequest {
			http.Error(w, "storage rejected payload", http.StatusBadGateway)
			return
		}
		writeJSON(w, shortenResponse{
			Code:     code,
			ShortURL: baseURL + "/" + code,
		})
	})

	log.Printf("api listening on :%s", port)
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
