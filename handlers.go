package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidAuth = errors.New("invalid username or password")

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateUser handles the /user/create endpoint. It should receive from KOReader
// a username and MD5 hashed password. If the user already exists, returns an error
// else, creates de user
func CreateUser(w http.ResponseWriter, r *http.Request, store *Store) {
	var req CreateUserRequest

	r.Body = http.MaxBytesReader(w, r.Body, 1024) // 1KB limit to prevent DoS

	// Decodes the request, returns if fails
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Checks if Username and Password are not empty. If empty assume the JSON
	// is nor correct
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Invalid body request: Malformed JSON", http.StatusBadRequest)
		return
	}

	// Checks if the provided username has valid characters
	var validUsername = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validUsername.MatchString(req.Username) {
		http.Error(w, "Invalid username", http.StatusBadRequest)
		return
	}

	// Checks if the username already exists
	store.mu.RLock()
	_, exists := store.Users[req.Username]
	store.mu.RUnlock()
	if exists {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	// Checks if password is not too short or too long
	// More of a sanity check because in the current implementation of KOReader
	// the password is never sent but a (insecure) MD5 hash
	if len(req.Password) > 72 || len(req.Password) < 8 {
		http.Error(w, "Password not meet requirements", http.StatusBadRequest)
		return
	}

	// Passing all the checks, creates a strong bcrypt hash of the password
	// (actually the MD5 hash) to store. Treating the MD5 as plaintext because
	// it is just a secure.
	keyBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Hash error", http.StatusInternalServerError)
		return
	}

	// Stores the new user
	store.mu.Lock()
	defer store.mu.Unlock()
	store.Users[req.Username] = User{
		Username: req.Username,
		Key:      string(keyBytes),
	}

	// Returns the success to the client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // Returns 201
	json.NewEncoder(w).Encode(map[string]string{"username": req.Username})
}

func AuthUser(w http.ResponseWriter, r *http.Request, store *Store) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024) // 1KB limit to prevent DoS

	_, err := requiresAuth(r, store)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Returns 200

	json.NewEncoder(w).Encode(map[string]string{
		"authorized": "OK",
	})

}

func UpdateProgress(w http.ResponseWriter, r *http.Request, store *Store) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024) // 1KB limit to prevent DoS
	authUser, err := requiresAuth(r, store)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	var reqProgress Progress

	err = json.NewDecoder(r.Body).Decode(&reqProgress)
	if err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if reqProgress.Document == "" || reqProgress.Progress == "" || reqProgress.Device == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	reqProgress.Timestamp = time.Now().Unix()

	store.mu.Lock()
	if store.Progresses[authUser] == nil {
		store.Progresses[authUser] = make(map[string]Progress)
	}
	store.Progresses[authUser][reqProgress.Document] = reqProgress
	store.mu.Unlock()

	// Returns the success to the client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Returns 200
	json.NewEncoder(w).Encode(map[string]any{
		"document":  reqProgress.Document,
		"timestamp": reqProgress.Timestamp,
	})

}

// requiresAuth should be called whenever the endpoint requires authentication
// the function will return the username if authenticated, and error if not
func requiresAuth(r *http.Request, store *Store) (string, error) {
	headerUser := r.Header.Get("x-auth-user")
	headerPass := r.Header.Get("x-auth-key")

	if headerPass == "" || headerUser == "" {
		return "", ErrInvalidAuth
	}

	store.mu.RLock()
	foundUser, exists := store.Users[headerUser]
	store.mu.RUnlock()

	if !exists {
		return "", ErrInvalidAuth
	}

	err := bcrypt.CompareHashAndPassword([]byte(foundUser.Key), []byte(headerPass))
	if err != nil {
		return "", ErrInvalidAuth
	}
	return headerUser, nil

}
