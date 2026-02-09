package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"test/internal/codec"
	"test/internal/store"
)

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

type storagePayload struct {
	Code   string `json:"code"`
	URL    string `json:"url"`
	UserID string `json:"user_id"`
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type linkResponse struct {
	Code     string `json:"code"`
	URL      string `json:"url"`
	ShortURL string `json:"short_url"`
	Clicks   int    `json:"clicks"`
}

type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]string
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]string)}
}

func (s *sessionStore) set(token, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = userID
}

func (s *sessionStore) get(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	userID, ok := s.sessions[token]
	return userID, ok
}

func (s *sessionStore) delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func main() {
	port := getenv("PORT", "8080")
	storageURL := strings.TrimRight(getenv("STORAGE_URL", "http://localhost:8082"), "/")
	baseURL := strings.TrimRight(getenv("BASE_URL", "http://localhost:8081"), "/")
	sessions := newSessionStore()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/", http.FileServer(http.Dir("web")))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req authRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if req.Email == "" || req.Password == "" {
			http.Error(w, "email and password required", http.StatusBadRequest)
			return
		}
		if _, err := fetchUserByEmail(storageURL, req.Email); err == nil {
			http.Error(w, "email already registered", http.StatusConflict)
			return
		}
		passwordHash, err := hashPassword(req.Password)
		if err != nil {
			http.Error(w, "failed to secure password", http.StatusInternalServerError)
			return
		}
		user, err := createUser(storageURL, req.Email, passwordHash)
		if err != nil {
			if errors.Is(err, store.ErrEmailExists) {
				http.Error(w, "email already registered", http.StatusConflict)
				return
			}
			http.Error(w, "failed to create user", http.StatusBadGateway)
			return
		}
		writeJSON(w, userResponse{ID: user.ID, Email: user.Email})
	})
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req authRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if req.Email == "" || req.Password == "" {
			http.Error(w, "email and password required", http.StatusBadRequest)
			return
		}
		user, err := fetchUserByEmail(storageURL, req.Email)
		if err != nil {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		if err := verifyPassword(user.PasswordHash, req.Password); err != nil {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		token, err := newToken()
		if err != nil {
			http.Error(w, "failed to create session", http.StatusInternalServerError)
			return
		}
		sessions.set(token, user.ID)
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(24 * time.Hour),
		})
		writeJSON(w, userResponse{ID: user.ID, Email: user.Email})
	})
	mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if cookie, err := r.Cookie("session"); err == nil {
			sessions.delete(cookie.Value)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/me", func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r, sessions, storageURL)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		writeJSON(w, userResponse{ID: user.ID, Email: user.Email})
	})
	mux.HandleFunc("/links", func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r, sessions, storageURL)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		links, err := listLinks(storageURL, user.ID)
		if err != nil {
			http.Error(w, "failed to fetch links", http.StatusBadGateway)
			return
		}
		responses := make([]linkResponse, 0, len(links))
		for _, link := range links {
			responses = append(responses, linkResponse{
				Code:     link.Code,
				URL:      link.URL,
				ShortURL: baseURL + "/" + link.Code,
				Clicks:   link.Clicks,
			})
		}
		writeJSON(w, responses)
	})
	mux.HandleFunc("/shorten", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		user, ok := currentUser(r, sessions, storageURL)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
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
		payload := storagePayload{Code: code, URL: req.URL, UserID: user.ID}
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
	mux.HandleFunc("/links/", func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r, sessions, storageURL)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		code := strings.TrimPrefix(r.URL.Path, "/links/")
		if code == "" {
			http.Error(w, "code required", http.StatusBadRequest)
			return
		}
		if err := deleteLink(storageURL, code, user.ID); err != nil {
			http.Error(w, "failed to delete link", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
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

func currentUser(r *http.Request, sessions *sessionStore, storageURL string) (store.User, bool) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return store.User{}, false
	}
	userID, ok := sessions.get(cookie.Value)
	if !ok {
		return store.User{}, false
	}
	user, err := fetchUser(storageURL, userID)
	if err != nil {
		return store.User{}, false
	}
	return user, true
}

func fetchUserByEmail(storageURL, email string) (store.User, error) {
	resp, err := http.Get(storageURL + "/users/by-email?email=" + url.QueryEscape(email))
	if err != nil {
		return store.User{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return store.User{}, errors.New("user not found")
	}
	var user store.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return store.User{}, err
	}
	return user, nil
}

func fetchUser(storageURL, id string) (store.User, error) {
	resp, err := http.Get(storageURL + "/users/" + id)
	if err != nil {
		return store.User{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return store.User{}, errors.New("user not found")
	}
	var user store.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return store.User{}, err
	}
	return user, nil
}

func createUser(storageURL, email, passwordHash string) (store.User, error) {
	payload := map[string]string{
		"email":         email,
		"password_hash": passwordHash,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return store.User{}, err
	}
	resp, err := http.Post(storageURL+"/users", "application/json", bytes.NewReader(body))
	if err != nil {
		return store.User{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return store.User{}, store.ErrEmailExists
	}
	if resp.StatusCode != http.StatusOK {
		return store.User{}, errors.New("failed to create user")
	}
	var user store.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return store.User{}, err
	}
	return user, nil
}

func listLinks(storageURL, userID string) ([]store.Link, error) {
	resp, err := http.Get(storageURL + "/links?user_id=" + url.QueryEscape(userID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch links")
	}
	var links []store.Link
	if err := json.NewDecoder(resp.Body).Decode(&links); err != nil {
		return nil, err
	}
	return links, nil
}

func deleteLink(storageURL, code, userID string) error {
	req, err := http.NewRequest(http.MethodDelete, storageURL+"/links/"+code+"?user_id="+url.QueryEscape(userID), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return errors.New("failed to delete")
	}
	return nil
}

func newToken() (string, error) {
	return codec.NewCode()
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := sha256.Sum256(append(salt, []byte(password)...))
	return base64.RawURLEncoding.EncodeToString(salt) + ":" + base64.RawURLEncoding.EncodeToString(hash[:]), nil
}

func verifyPassword(storedHash, password string) error {
	parts := strings.Split(storedHash, ":")
	if len(parts) != 2 {
		return errors.New("invalid hash")
	}
	salt, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return err
	}
	expected, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return err
	}
	hash := sha256.Sum256(append(salt, []byte(password)...))
	if !bytes.Equal(hash[:], expected) {
		return errors.New("invalid password")
	}
	return nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
