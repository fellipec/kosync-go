// kosync-go is a Go implementation of the KOReader Sync Server.
// It keeps everything as simple as possible, avoiding Redis server, being
// just a single executable that can run without any config files.
package main

import (
	"context"
	"crypto/tls"
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
	The original Koreader Sync Server listens on port 7200 by default for
	TLS connections and on port 17200 for plain HTTP.

	This implementation mimics that behavior. When running without any parameter,
	kosync-go will generate a self-signed TLS certificate and starts a web
	server on port 7200.

	It has the same endpoints and behave likes the original Koreader Sync Server
	to the clients. The server keeps its data in memory, dumping them to a JSON
	file every 5 minutes, if there are changes. The default file is called
	kosync.json and it will be created in the actual work folder.

	For more information refer to the readme.md file.
*/

// Loggers for the program
var LogInfo = log.New(os.Stdout, "INFO: ", log.LstdFlags)
var LogError = log.New(os.Stderr, "ERROR: ", log.LstdFlags)

var Version = "dev"

// main defines the endpoints handlers, process command line parameters, deal
// with signals from the system and periodically save the JSON file when changes
// occur.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// CLI parameters
	disableNewUsers := flag.Bool("disable-new-users", false, "disable creation of new users")
	storeFile := flag.String("store-file", "kosync.json", "path to store the JSON data file")
	insecure := flag.Bool("insecure", false, "run without TLS (for reverse proxy use)")
	port := flag.Int("port", 0, "port to listen on (default: 7200 for TLS, 17200 for insecure)")
	certFile := flag.String("cert", "", "path to TLS certificate file")
	keyFile := flag.String("key", "", "path to TLS key file")
	listenAddr := flag.String("listen-addr", "", "which address the server will listen for connections")
	versionFlag := flag.Bool("version", false, "displays the version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println("kosync-go ", Version)
		return
	}

	LogInfo.Printf("Starting kosync-go version %s", Version)

	// Makes sure if no port is defined to use the defaults
	if *port == 0 {
		if *insecure {
			*port = 17200 // Default port for plain text
		} else {
			*port = 7200 // Default port for TLS
		}
	}

	// Initializes a new store object
	mainStore, err := LoadStore(*storeFile)
	if err != nil {
		LogError.Panicln("Couldn't initialize data store: " + err.Error())
	}
	LogInfo.Printf("Loaded config file %s", *storeFile)

	// Handles /healthcheck, which just returns State: OK for troubleshooting purposes.
	http.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "{\"state\":\"OK\"}")
		LogInfo.Println("Health checked!")
	})

	// Handles /users/create endpoint, if the --disable-new-users option is not in use
	http.HandleFunc("/users/create", func(w http.ResponseWriter, r *http.Request) {
		if *disableNewUsers {
			http.Error(w, "Can't create new user", http.StatusForbidden)
			ip := GetIP(r)
			LogError.Printf("ALERT: Attempt to create user while --disable-new-users is active | IP=%s", ip)
			return
		}
		CreateUser(w, r, mainStore)
	})

	// Handlers for /users/auth and //syncs/progress
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
		Addr:         fmt.Sprintf("%s:%d", *listenAddr, *port),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		ErrorLog:     LogError,
	}

	// Runs the HTTP listener, obeying the CLI parameters
	go func() {
		if *insecure {
			err := srv.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				LogError.Panicln(err)
			}
		} else if *certFile != "" && *keyFile != "" {
			err := srv.ListenAndServeTLS(*certFile, *keyFile)
			if err != nil && err != http.ErrServerClosed {
				LogError.Panicln(err)
			}
		} else {
			selfCert, err := generateSelfSigned()
			if err != nil {
				LogError.Panicln("Error creating self-signed certificate", err)
			}
			srv.TLSConfig = &tls.Config{
				Certificates: []tls.Certificate{selfCert},
			}
			err = srv.ListenAndServeTLS("", "")
			if err != nil && err != http.ErrServerClosed {
				LogError.Panicln(err)
			}
		}
	}()
	if *insecure {
		LogInfo.Printf("kosync-go listening insecurely on port %d", *port)
	} else {
		LogInfo.Printf("kosync-go listening with TLS on port %d", *port)
	}

	// Saves data to file ever 5 minutes, if there is data to be saved
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if mainStore.dirty.Load() {
					LogInfo.Println("Data changed, saving state")
					err = SaveStore(mainStore, *storeFile)
					if err != nil {
						LogError.Panicln(err.Error())
					}
					mainStore.dirty.Store(false)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Gracefully ends the server on SIGTERM and SIGINT
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

// Gets the IP address of the remote client, for logging purposes.
func GetIP(r *http.Request) string {
	var ip string

	// Direct connections
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr // fallback
	}

	// With reverse proxy
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ip = xff + " (via " + ip + ")"
	}
	return ip
}
