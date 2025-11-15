package prometheus

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	once    sync.Once
	enabled atomic.Bool
)

// A Config is a prometheus config.
type Config struct {
	Host string
	Port int
	Path string
}

// Enabled reports whether Prometheus metrics are enabled.
func Enabled() bool {
	return enabled.Load()
}

// Enable enables Prometheus metrics.
func Enable() {
	enabled.Store(true)
}

// Start starts the Prometheus metrics server.
func Start(c Config) {
	defaultConfig(&c)
	once.Do(func() {
		Enable()
		go func() {
			http.Handle(c.Path, promhttp.Handler())
			addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
			if err := http.ListenAndServe(addr, nil); err != nil {
				log.Fatalf("prometheus: failed to start prometheus metrics server: %v", err)
			}
		}()
	})
}

func defaultConfig(conf *Config) {
	if conf.Path == "" {
		conf.Path = "/metrics"
	}
	if conf.Port == 0 {
		conf.Port = 9101
	}
}
