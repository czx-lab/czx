package metrics

import (
	"errors"

	prom "github.com/prometheus/client_golang/prometheus"
)

type promGauge struct {
	gauge *prom.GaugeVec
}

var _ Gauge = (*promGauge)(nil)

func NewGauge(conf *VectorOption) Gauge {
	if conf == nil {
		return nil
	}
	vec := prom.NewGaugeVec(prom.GaugeOpts{
		Namespace: conf.Namespace,
		Subsystem: conf.Subsystem,
		Name:      conf.Name,
		Help:      conf.Help,
	}, conf.Labels)
	prom.MustRegister(vec)
	gv := &promGauge{
		gauge: vec,
	}
	return gv
}

// Add implements Gauge.
func (p *promGauge) Add(delta float64, labels ...string) {
	update(func() {
		p.gauge.WithLabelValues(labels...).Add(delta)
	})
}

// Close implements Gauge.
func (p *promGauge) Close() error {
	if prom.Unregister(p.gauge) {
		return nil
	}
	return errors.New("failed to unregister gauge metric")
}

// Dec implements Gauge.
func (p *promGauge) Dec(labels ...string) {
	update(func() {
		p.gauge.WithLabelValues(labels...).Dec()
	})
}

// Inc implements Gauge.
func (p *promGauge) Inc(labels ...string) {
	update(func() {
		p.gauge.WithLabelValues(labels...).Inc()
	})
}

// Set implements Gauge.
func (p *promGauge) Set(value float64, labels ...string) {
	update(func() {
		p.gauge.WithLabelValues(labels...).Set(value)
	})
}

// Sub implements Gauge.
func (p *promGauge) Sub(delta float64, labels ...string) {
	update(func() {
		p.gauge.WithLabelValues(labels...).Sub(delta)
	})
}
