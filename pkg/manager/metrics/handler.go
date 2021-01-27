package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// GalaxySubsystem metric prefix name
	GalaxySubsystem = "galaxy"
)

// metricHandler serve /metrics
type metricHandler struct {
}

// NewHandler build a handler for metrics
func NewHandler() http.Handler {
	return &metricHandler{}
}

// ServeHTTP implements handler
func (s *metricHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	Metrics()

	// serve prometheus metrics
	promhttp.Handler().ServeHTTP(w, req)
}

// Metrics all runtime status metrics
func Metrics() {
	//TODO: runtime status monitor metrics
}
