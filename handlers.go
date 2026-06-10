package main

import (
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateUser handles the /user/create endpoint. It should receive from KOReader
// a username and MD5 hashed password. If the user already exists, returns an error
// else, creates de user
func CreateUser(w http.ResponseWriter, r *http.Request, store *Store) {
	var req CreateUserRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Invalid body request: Malformed JSON", http.StatusBadRequest)
		return
	}

	store.mu.RLock()
	_, exists := store.Users[req.Username]
	store.mu.RUnlock()
	if exists {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	keyBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Hash error", http.StatusInternalServerError)
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	store.Users[req.Username] = User{
		Username: req.Username,
		Key:      string(keyBytes),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // Returns 201
	json.NewEncoder(w).Encode(map[string]string{"username": req.Username})
	return
}
