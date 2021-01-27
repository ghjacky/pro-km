package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// HTTPRequests tracks the number of the http requests received since the server started.
	HTTPRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: GalaxySubsystem,
			Name:      "http_requests_total",
			Help:      "Number of the http requests received since the server started",
		},
		// server_type aims to differentiate the readonly server and the readwrite server.
		// long_running marks whether the request is long-running or not.
		// Currently, long-running requests include exec/attach/portforward/debug.
		[]string{"method", "path"},
	)
	// HTTPRequestsDuration tracks the duration in seconds to serve http requests.
	HTTPRequestsDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: GalaxySubsystem,
			Name:      "http_requests_duration_seconds",
			Help:      "Duration in seconds to serve http requests",
			// Use DefBuckets for now, will customize the buckets if necessary.
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	// HTTPInflightRequests tracks the number of the inflight http requests.
	HTTPInflightRequests = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: GalaxySubsystem,
			Name:      "http_inflight_requests",
			Help:      "Number of the inflight http requests",
		},
		[]string{"method", "path"},
	)
)

// Register all metrics.
func init() {
	prometheus.MustRegister(HTTPRequests)
	prometheus.MustRegister(HTTPRequestsDuration)
	prometheus.MustRegister(HTTPInflightRequests)
}

// SinceInSeconds gets the time since the specified start in seconds.
func SinceInSeconds(start time.Time) float64 {
	return time.Since(start).Seconds()
}
