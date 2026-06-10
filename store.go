package main

import (
	"sync"
)

// User represents a registered user with a username and a hashed password key.
type User struct {
	Username string `json:"username"`
	Key      string `json:"key"` // a bcrypt hash of the received password send from KOReader
}

// Progress represents a book progress object, as received from the KOReader.
type Progress struct {
	Progress   string  `json:"progress"`
	Percentage float64 `json:"percentage"`
	Device     string  `json:"device"`
	DeviceID   string  `json:"device_id"`
	Timestamp  int64   `json:"timestamp"`
}

// Store is the in memory a list of users and the book progresses of each users' book
// The Progress map is structured as user -> book hash -> Progress object.
type Store struct {
	mu         sync.RWMutex
	Users      map[string]User
	Progresses map[string]map[string]Progress
}

// NewStore creates and initializes a new Store object.
func NewStore() Store {
	var s Store
	s.Users = make(map[string]User)
	s.Progresses = make(map[string]map[string]Progress)
	return s
}
