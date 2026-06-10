// kosync-go is a Go implementation of the KOReader Sync Server.
// It keeps everything as simple as possible, avoiding Redis server, being
// just a single executable that can run without any config files.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
)

/*
Port is the default port the server listens on.

	The original Koreader Sync Server listens on port 7200 by default for
	TLS connections and on port 17200 for plain HTTP.
*/
const Port = 17200

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	storeFile := flag.String("store-file", "kosync.json", "path to store the JSON data file")
	flag.Parse()

	// Initializes a new store object

	mainStore, err := LoadStore(*storeFile)
	if err != nil {
		panic("Couldn't initialize data store: " + err.Error())
	}

	// Handles /healthcheck, which just returns State: OK for troubleshooting purposes.
	http.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "{\"state\":\"OK\"}")
	})

	http.HandleFunc("/users/create", func(w http.ResponseWriter, r *http.Request) {
		CreateUser(w, r, mainStore)
	})

	http.HandleFunc("/users/auth", func(w http.ResponseWriter, r *http.Request) {
		AuthUser(w, r, mainStore)
	})

	http.HandleFunc("/syncs/progress", func(w http.ResponseWriter, r *http.Request) {
		UpdateProgress(w, r, mainStore)
	})

	http.HandleFunc("GET /syncs/progress/{document}", func(w http.ResponseWriter, r *http.Request) {
		GetProgress(w, r, mainStore)
	})

	// Creates the HTTP listener
	addr := fmt.Sprintf(":%d", Port)
	fmt.Printf("kosync-go listening on port %d\n", Port)
	go http.ListenAndServe(addr, nil)

	<-ctx.Done()
	fmt.Println("Saving state")
	err = SaveStore(mainStore, *storeFile)
	if err != nil {
		panic(err.Error())
	}

}
