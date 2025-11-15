package metrics

import (
	"errors"

	prom "github.com/prometheus/client_golang/prometheus"
)

type promCounter struct {
	counter *prom.CounterVec
}

var _ Counter = (*promCounter)(nil)

func NewCounter(conf *VectorOption) Counter {
	if conf == nil {
		return nil
	}
	vec := prom.NewCounterVec(prom.CounterOpts{
		Namespace: conf.Namespace,
		Subsystem: conf.Subsystem,
		Name:      conf.Name,
		Help:      conf.Help,
	}, conf.Labels)
	prom.MustRegister(vec)
	cv := &promCounter{
		counter: vec,
	}
	return cv
}

// Add implements Counter.
func (p *promCounter) Add(delta float64, labels ...string) {
	update(func() {
		p.counter.WithLabelValues(labels...).Add(delta)
	})
}

// Close implements Counter.
func (p *promCounter) Close() error {
	if prom.Unregister(p.counter) {
		return nil
	}
	return errors.New("failed to unregister counter metric")
}

// Inc implements Counter.
func (p *promCounter) Inc(labels ...string) {
	update(func() {
		p.counter.WithLabelValues(labels...).Inc()
	})
}
