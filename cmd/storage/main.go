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
	Code   string `json:"code"`
	URL    string `json:"url"`
	UserID string `json:"user_id"`
}

type userRequest struct {
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
}

func main() {
	port := getenv("PORT", "8082")
	svc := store.NewMemoryStore()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req userRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if req.Email == "" || req.PasswordHash == "" {
			http.Error(w, "email and password required", http.StatusBadRequest)
			return
		}
		user, err := svc.CreateUser(req.Email, req.PasswordHash)
		if err != nil {
			if errors.Is(err, store.ErrEmailExists) {
				http.Error(w, "email already registered", http.StatusConflict)
				return
			}
			http.Error(w, "failed to create user", http.StatusInternalServerError)
			return
		}
		writeJSON(w, user)
	})
	mux.HandleFunc("/users/by-email", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		email := r.URL.Query().Get("email")
		if email == "" {
			http.Error(w, "email required", http.StatusBadRequest)
			return
		}
		user, err := svc.GetUserByEmail(email)
		if err != nil {
			if errors.Is(err, store.ErrUserNotFound) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, "failed to fetch user", http.StatusInternalServerError)
			return
		}
		writeJSON(w, user)
	})
	mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/users/")
		if id == "" {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		user, err := svc.GetUser(id)
		if err != nil {
			if errors.Is(err, store.ErrUserNotFound) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, "failed to fetch user", http.StatusInternalServerError)
			return
		}
		writeJSON(w, user)
	})
	mux.HandleFunc("/links", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req linkRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid payload", http.StatusBadRequest)
				return
			}
			if req.Code == "" || req.URL == "" || req.UserID == "" {
				http.Error(w, "code, url, and user_id required", http.StatusBadRequest)
				return
			}
			svc.SaveLink(store.Link{Code: req.Code, URL: req.URL, UserID: req.UserID})
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			userID := r.URL.Query().Get("user_id")
			if userID == "" {
				http.Error(w, "user_id required", http.StatusBadRequest)
				return
			}
			links := svc.ListLinksByUser(userID)
			writeJSON(w, links)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
	})
	mux.HandleFunc("/links/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/links/")
		if path == "" {
			http.Error(w, "code required", http.StatusBadRequest)
			return
		}
		if strings.HasSuffix(path, "/click") {
			code := strings.TrimSuffix(path, "/click")
			code = strings.TrimSuffix(code, "/")
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if err := svc.IncrementClick(code); err != nil {
				if errors.Is(err, store.ErrNotFound) {
					http.Error(w, "not found", http.StatusNotFound)
					return
				}
				http.Error(w, "failed to update", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		code := strings.TrimSuffix(path, "/")
		switch r.Method {
		case http.MethodGet:
			link, err := svc.GetLink(code)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					http.Error(w, "not found", http.StatusNotFound)
					return
				}
				http.Error(w, "failed to fetch", http.StatusInternalServerError)
				return
			}
			writeJSON(w, link)
		case http.MethodDelete:
			userID := r.URL.Query().Get("user_id")
			if userID == "" {
				http.Error(w, "user_id required", http.StatusBadRequest)
				return
			}
			if err := svc.DeleteLink(code, userID); err != nil {
				if errors.Is(err, store.ErrNotFound) {
					http.Error(w, "not found", http.StatusNotFound)
					return
				}
				if errors.Is(err, store.ErrUnauthorized) {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
				http.Error(w, "failed to delete", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
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
