package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

type linkResponse struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

func main() {
	port := getenv("PORT", "8081")
	storageURL := strings.TrimRight(getenv("STORAGE_URL", "http://localhost:8082"), "/")

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		code := strings.TrimPrefix(r.URL.Path, "/")
		if code == "" {
			http.Error(w, "code required", http.StatusBadRequest)
			return
		}
		link, err := fetchLink(storageURL, code)
		if err != nil {
			http.Error(w, "link not found", http.StatusNotFound)
			return
		}
		http.Redirect(w, r, link.URL, http.StatusFound)
	})

	log.Printf("redirect listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func fetchLink(storageURL, code string) (linkResponse, error) {
	resp, err := http.Get(fmt.Sprintf("%s/links/%s", storageURL, code))
	if err != nil {
		return linkResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return linkResponse{}, fmt.Errorf("not found")
	}
	var link linkResponse
	if err := json.NewDecoder(resp.Body).Decode(&link); err != nil {
		return linkResponse{}, err
	}
	return link, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
