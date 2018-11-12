package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	addr    = ":8080"
	dataDir = "./data"
)

func ping(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("pong\n"))
}

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s\n", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func main() {
	// router handlers
	http.HandleFunc("/ping", ping)
	http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(dataDir))))
	http.Handle("/metrics", promhttp.Handler())

	log.Println("Waiting for requests on ", addr)
	log.Fatal(http.ListenAndServe(addr, logRequest(http.DefaultServeMux)))
}
