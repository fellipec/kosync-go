// kosync-go is a Go implementation of the KOReader Sync Server.
// It keeps everything as simple as possible, avoiding Redis server, being
// just a single executable that can run without any config files.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

/*
Port is the default port the server listens on.

	The original Koreader Sync Server listens on port 7200 by default for
	TLS connections and on port 17200 for plain HTTP.
*/
const Port = 17200

var LogInfo = log.New(os.Stdout, "INFO: ", log.LstdFlags)
var LogError = log.New(os.Stderr, "ERROR:", log.LstdFlags)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// CLI parameters
	disableNewUsers := flag.Bool("disable-new-users", false, "disable creation of new users")
	storeFile := flag.String("store-file", "kosync.json", "path to store the JSON data file")
	flag.Parse()

	// Initializes a new store object

	mainStore, err := LoadStore(*storeFile)
	if err != nil {
		LogError.Panicln("Couldn't initialize data store: " + err.Error())
	}

	// Handles /healthcheck, which just returns State: OK for troubleshooting purposes.
	http.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "{\"state\":\"OK\"}")
		LogInfo.Println("Health checked!")
	})

	http.HandleFunc("/users/create", func(w http.ResponseWriter, r *http.Request) {
		if *disableNewUsers {
			http.Error(w, "Can't create new user", http.StatusForbidden)
			ip := GetIP(r)
			LogError.Printf("ALERT: Attempt to create user while --disable-new-users is active | IP=%s", ip)
			return
		}
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

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", Port),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			LogError.Panicln(err)
		}
	}()
	LogInfo.Printf("kosync-go listening on port %d", Port)

	<-ctx.Done()
	LogInfo.Println("Stopping server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)

	LogInfo.Println("Saving state")
	err = SaveStore(mainStore, *storeFile)
	if err != nil {
		LogError.Panicln(err.Error())
	}
	LogInfo.Println("State saved, exiting")

}

func GetIP(r *http.Request) string {
	// With reverse proxy
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}

	// Direct connections
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr // fallback
	}
	return ip
}
