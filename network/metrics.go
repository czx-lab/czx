package network

import "time"

// ServerMetrics defines the interface for server metrics tracking.
type ServerMetrics interface {
	// Connection metrics
	// Increment the count of active connections
	IncConns()
	// Decrement the count of active connections
	DecConns()
	// Increment the total number of connections made
	IncTotalConns()
	// Increment the count of failed connection attempts
	IncFailedConns()
	// Observe the duration of a connection
	ObserveConnDuration(duration time.Duration)

	// Data transfer metrics
	// Add the number of bytes sent
	AddSentBytes(bytes int)
	// Add the number of bytes received
	AddReceivedBytes(bytes int)

	// Error metrics
	// Increment the count of read errors encountered
	IncReadErrors()
	// Increment the count of write errors encountered
	IncWriteErrors()

	// Shutdown the metrics tracking system
	Close() error
}

type NoopServerMetrics struct{}

// AddReceivedBytes implements ServerMetrics.
func (n *NoopServerMetrics) AddReceivedBytes(bytes int) {}

// AddSentBytes implements ServerMetrics.
func (n *NoopServerMetrics) AddSentBytes(bytes int) {}

// Close implements ServerMetrics.
func (n *NoopServerMetrics) Close() error { return nil }

// DecConns implements ServerMetrics.
func (n *NoopServerMetrics) DecConns() {}

// IncConns implements ServerMetrics.
func (n *NoopServerMetrics) IncConns() {}

// IncFailedConns implements ServerMetrics.
func (n *NoopServerMetrics) IncFailedConns() {}

// IncReadErrors implements ServerMetrics.
func (n *NoopServerMetrics) IncReadErrors() {}

// IncTotalConns implements ServerMetrics.
func (n *NoopServerMetrics) IncTotalConns() {}

// IncWriteErrors implements ServerMetrics.
func (n *NoopServerMetrics) IncWriteErrors() {}

// ObserveConnDuration implements ServerMetrics.
func (n *NoopServerMetrics) ObserveConnDuration(duration time.Duration) {}

var _ ServerMetrics = (*NoopServerMetrics)(nil)
