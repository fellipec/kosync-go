package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// User represents a registered user with a username and a hashed password key.
type User struct {
	Username string `json:"username"`
	Key      string `json:"key"` // a bcrypt hash of the received password send from KOReader
}

// Progress represents a book progress object, as received from the KOReader.
type Progress struct {
	Document   string  `json:"document"`
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
	dirty      atomic.Bool
	Users      map[string]User
	Progresses map[string]map[string]Progress
}

// NewStore creates and initializes a new Store object.
func NewStore() *Store {
	s := &Store{}
	s.Users = make(map[string]User)
	s.Progresses = make(map[string]map[string]Progress)
	return s
}

func SaveStore(store *Store, path string) error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, "kosync")
	if err != nil {
		return err
	}

	err = json.NewEncoder(f).Encode(store)
	if err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	err = os.Rename(f.Name(), path)
	if err != nil {
		return err
	}
	return nil
}

func LoadStore(path string) (*Store, error) {
	s := NewStore()

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, err
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(s)
	if err != nil {
		return s, err
	}

	if s.Users == nil {
		return s, fmt.Errorf("invalid store: missing Users map")
	}
	if s.Progresses == nil {
		return s, fmt.Errorf("invalid store: missing Progresses map")
	}

	return s, nil

}
