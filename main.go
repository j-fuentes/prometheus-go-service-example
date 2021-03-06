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
	addr          = ":8080"
	dataDir       = "./data"
	webDir        = "./web"
	promNamespace = "quiz"
)

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s\n", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func handleError(status int, msg string, w http.ResponseWriter) error {
	w.WriteHeader(status)
	w.Write([]byte(msg))
	return fmt.Errorf(msg)
}

func goToSleepIfNeeded(w http.ResponseWriter, r *http.Request) error {
	s := r.URL.Query().Get("sleep")
	if s != "" {
		i, err := strconv.Atoi(s)
		if err != nil {
			return handleError(400, "only integers allowed with 'sleep'", w)
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
		return handleError(i, fmt.Sprintf("status forced to %d\n", i), w)
	}
	return nil
}

// handlers
func empty(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte{})
}

func ping(w http.ResponseWriter, r *http.Request) {
	if err := goToSleepIfNeeded(w, r); err != nil {
		return
	}

	if err := forceStatusIfNeeded(w, r); err != nil {
		return
	}

	w.Write([]byte("pong\n"))
}

func file(w http.ResponseWriter, r *http.Request) {

}

func main() {
	// RED metrics
	inFlightReqGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: promNamespace,
		Name:      "api_in_flight_requests",
		Help:      "A gauge of requests currently being served by the wrapped handler.",
	})

	reqCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: promNamespace,
			Name:      "api_requests_total",
			Help:      "A counter for requests.",
		},
		[]string{"code", "method"},
	)

	reqDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: promNamespace,
			Name:      "api_requests_duration_seconds",
			Help:      "A histogram of latencies.",
			Buckets:   []float64{.25, .5, 1, 2.5, 5, 10},
		},
		[]string{"code", "method"},
	)

	// quiz metrics
	visitCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: promNamespace,
			Name:      "visit_counter",
			Help:      "A counter for visits to the quiz.",
		},
		[]string{},
	)

	answerCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: promNamespace,
			Name:      "quiz_answer_event",
			Help:      "A counter for answers to the quiz.",
		},
		[]string{"question", "result"},
	)

	answerHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: promNamespace,
			Name:      "quiz_result",
			Help:      "A histogram of quiz results.",
			Buckets:   []float64{0, 1, 2, 3, 4, 5, 6},
		},
		[]string{},
	)

	prometheus.MustRegister(inFlightReqGauge, reqCounter, reqDuration, visitCounter, answerCounter, answerHistogram)

	// instrumentation chains
	instrumentHandler := func(handler http.Handler) http.Handler {
		return promhttp.InstrumentHandlerInFlight(
			inFlightReqGauge,
			promhttp.InstrumentHandlerDuration(
				reqDuration,
				promhttp.InstrumentHandlerCounter(
					reqCounter,
					handler,
				),
			),
		)
	}

	// router handlers
	http.Handle("/", instrumentHandler(presentQuiz(&quizMetrics{
		visitCounter: visitCounter,
	})))
	http.Handle("/answer", instrumentHandler(answerQuiz(&quizMetrics{
		answerCounter:   answerCounter,
		answerHistogram: answerHistogram,
	})))
	http.Handle("/images/", instrumentHandler(http.StripPrefix("/images/", http.FileServer(http.Dir(dataDir)))))
	http.Handle("/ping", instrumentHandler(http.HandlerFunc(ping)))
	http.Handle("/metrics", promhttp.Handler())

	log.Println("Waiting for requests on ", addr)
	log.Fatal(http.ListenAndServe(addr, logRequest(http.DefaultServeMux)))
}
