package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	addr    = ":8080"
	dataDir = "./data"
)

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s\n", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func goToSleepIfNeeded(w http.ResponseWriter, r *http.Request) error {
	s := r.URL.Query().Get("sleep")
	if s != "" {
		i, err := strconv.Atoi(s)
		if err != nil {
			msg := "only integers allowed with 'sleep'"
			w.WriteHeader(400)
			w.Write([]byte(msg))
			return fmt.Errorf(msg)
		}

		time.Sleep(time.Duration(i) * time.Millisecond)
	}
	return nil
}

func forceStatusIfNeeded(w http.ResponseWriter, r *http.Request) error {
	s := r.URL.Query().Get("forceStatus")
	if s != "" {
		i, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		msg := fmt.Sprintf("status forced to %d\n", i)
		w.WriteHeader(i)
		w.Write([]byte(msg))
		return fmt.Errorf(msg)
	}
	return nil
}

// handlers
func ping(w http.ResponseWriter, r *http.Request) {
	if err := goToSleepIfNeeded(w, r); err != nil {
		return
	}

	if err := forceStatusIfNeeded(w, r); err != nil {
		return
	}

	w.Write([]byte("pong\n"))
}

func main() {
	// custom metrics
	inFlightGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "api_in_flight_requests",
		Help: "A gauge of requests currently being served by the wrapped handler.",
	})

	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_requests_total",
			Help: "A counter for requests to the wrapped handler.",
		},
		[]string{"code", "method"},
	)

	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_requests_duration_seconds",
			Help:    "A histogram of latencies",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10},
		},
		[]string{"code", "method"},
	)

	prometheus.MustRegister(inFlightGauge, counter, duration)

	// instrumentation chains
	pingChain := promhttp.InstrumentHandlerInFlight(
		inFlightGauge,
		promhttp.InstrumentHandlerDuration(
			duration,
			promhttp.InstrumentHandlerCounter(
				counter,
				http.HandlerFunc(ping),
			),
		),
	)

	// router handlers
	http.Handle("/ping", pingChain)
	http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(dataDir))))
	http.Handle("/metrics", promhttp.Handler())

	log.Println("Waiting for requests on ", addr)
	log.Fatal(http.ListenAndServe(addr, logRequest(http.DefaultServeMux)))
}
