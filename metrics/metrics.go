package metrics

import "github.com/czx-lab/czx/prometheus"

type (
	// VectorOption defines options for creating metric vectors.
	VectorOption struct {
		Namespace string
		Subsystem string
		Name      string
		Help      string
		Labels    []string
	}
	// Metrics defines the interface for metrics collection and reporting.
	Metrics interface {
		// Close closes the metrics instance.
		Close() error
	}
	// Counter defines the interface for a counter metric.
	Counter interface {
		Metrics
		Inc(labels ...string)
		Add(delta float64, labels ...string)
	}
	// Gauge defines the interface for a gauge metric.
	Gauge interface {
		Metrics
		Set(value float64, labels ...string)
		Inc(labels ...string)
		Dec(labels ...string)
		Add(delta float64, labels ...string)
		Sub(delta float64, labels ...string)
	}
	// Histogram defines the interface for a histogram metric.
	Histogram interface {
		Metrics
		Observe(value float64, labels ...string)
	}
	// Summary defines the interface for a summary metric.
	Summary interface {
		Metrics
		Observe(value float64, labels ...string)
	}
)

func update(fn func()) {
	if !prometheus.Enabled() {
		return
	}
	fn()
}
