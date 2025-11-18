package metrics

import (
	"time"

	"github.com/czx-lab/czx/metrics"
	"github.com/czx-lab/czx/network"
)

type (
	// SvrMetrics holds various metrics related to server performance and operations
	SvrMetrics struct {
		// connection metrics
		activeConns  metrics.Gauge
		totalConns   metrics.Counter
		connDuration metrics.Histogram

		// message metrics
		receivedBytes metrics.Counter
		sentBytes     metrics.Counter

		// error metrics
		errors metrics.Counter
	}
	// SvrMetricsConf defines the configuration for server metrics
	SvrMetricsConf struct {
		Namespace string
		Subsystem string
	}
)

var _ network.ServerMetrics = (*SvrMetrics)(nil)

// NewSvrMetrics creates and initializes a new SvrMetrics instance based on the provided configuration.
// It sets up various gauges, counters, and histograms to monitor server performance.
// The metrics include active connections, total connections, bytes received/sent, connection duration, and error counts.
// These metrics can be used for monitoring and analyzing server behavior over time.
// Returns a pointer to the initialized SvrMetrics instance.
func NewSvrMetrics(conf SvrMetricsConf) *SvrMetrics {
	return &SvrMetrics{
		activeConns: metrics.NewGauge(&metrics.VectorOption{
			Namespace: conf.Namespace,
			Subsystem: conf.Subsystem,
			Name:      "active_connections",
			Help:      "current number of active connections",
		}),
		totalConns: metrics.NewCounter(&metrics.VectorOption{
			Namespace: conf.Namespace,
			Subsystem: conf.Subsystem,
			Name:      "connections_total",
			Help:      "total number of connections",
		}),
		receivedBytes: metrics.NewCounter(&metrics.VectorOption{
			Namespace: conf.Namespace,
			Subsystem: conf.Subsystem,
			Name:      "received_bytes_total",
			Help:      "total bytes received",
		}),
		sentBytes: metrics.NewCounter(&metrics.VectorOption{
			Namespace: conf.Namespace,
			Subsystem: conf.Subsystem,
			Name:      "sent_bytes_total",
			Help:      "total bytes sent",
		}),
		connDuration: metrics.NewHistogram(&metrics.HistogramVecOpts{
			VectorOption: metrics.VectorOption{
				Namespace: conf.Namespace,
				Subsystem: conf.Subsystem,
				Name:      "connection_duration_seconds",
				Help:      "connection duration in seconds",
			},
			Buckets: []float64{1, 10, 60, 300, 600, 1800, 3600},
		}),
		errors: metrics.NewCounter(&metrics.VectorOption{
			Namespace: conf.Namespace,
			Subsystem: conf.Subsystem,
			Name:      "errors_total",
			Help:      "Total errors by type",
			Labels:    []string{"type"}, // read/write/parse/upgrade/connect
		}),
	}
}

// AddReceivedBytes implements network.ServerMetrics.
func (s *SvrMetrics) AddReceivedBytes(bytes int) {
	s.receivedBytes.Add(float64(bytes))
}

// AddSentBytes implements network.ServerMetrics.
func (s *SvrMetrics) AddSentBytes(bytes int) {
	s.sentBytes.Add(float64(bytes))
}

// Close implements network.ServerMetrics.
func (s *SvrMetrics) Close() error {
	return nil
}

// DecConns implements network.ServerMetrics.
func (s *SvrMetrics) DecConns() {
	s.activeConns.Dec()
}

// IncConns implements network.ServerMetrics.
func (s *SvrMetrics) IncConns() {
	s.activeConns.Inc()
}

// IncFailedConns implements network.ServerMetrics.
func (s *SvrMetrics) IncFailedConns() {
	s.errors.Inc("connect")
}

// IncReadErrors implements network.ServerMetrics.
func (s *SvrMetrics) IncReadErrors() {
	s.errors.Inc("read")
}

// IncTotalConns implements network.ServerMetrics.
func (s *SvrMetrics) IncTotalConns() {
	s.totalConns.Inc()
}

// IncWriteErrors implements network.ServerMetrics.
func (s *SvrMetrics) IncWriteErrors() {
	s.errors.Inc("write")
}

// ObserveConnDuration implements network.ServerMetrics.
func (s *SvrMetrics) ObserveConnDuration(duration time.Duration) {
	s.connDuration.Observe(duration.Seconds())
}
