// kosync-go is a Go implementation of the KOReader Sync Server.
// It keeps everything as simple as possible, avoiding Redis server, being
// just a single executable that can run without any config files.
package main

import (
	"fmt"
	"net/http"
)

/*
Port is the default port the server listens on.

	The original Koreader Sync Server listens on port 7200 by default for
	TLS connections and on port 17200 for plain HTTP.
*/
const Port = 17200

func main() {
	// Initializes a new store object
	mainStore := NewStore()

	// Handles /healthcheck, which just returns State: OK for troubleshooting purposes.
	http.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "{\"state\":\"OK\"}")
	})

	http.HandleFunc("/users/create", func(w http.ResponseWriter, r *http.Request) {
		CreateUser(w, r, &mainStore)
	})

	// Creates the HTTP listener
	addr := fmt.Sprintf(":%d", Port)
	fmt.Printf("kosync-go listening on port %d\n", Port)
	http.ListenAndServe(addr, nil)

}
