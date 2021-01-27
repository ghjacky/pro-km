package conns

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// GalaxySubsystem metric prefix name
	GalaxySubsystem = "galaxy"
)

var (
	// GRPCConnNumber metric of total grpc connection number
	GRPCConnNumber = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Subsystem: GalaxySubsystem,
			Name:      "grpc_conn_number",
			Help:      "Number of grpc connections",
		},
	)
)

// Register all metrics
func init() {
	prometheus.MustRegister(GRPCConnNumber)
}
