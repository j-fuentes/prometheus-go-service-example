package main

import (
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

// handlers
func ping(w http.ResponseWriter, r *http.Request) {
	s := r.URL.Query().Get("sleep")
	if s != "" {
		i, err := strconv.Atoi(s)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte("only integers allowed with 'sleep'"))
			return
		}

		time.Sleep(time.Duration(i) * time.Millisecond)
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
