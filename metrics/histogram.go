package metrics

import (
	"errors"

	prom "github.com/prometheus/client_golang/prometheus"
)

type (
	// A HistogramVecOpts is a histogram vector options.
	HistogramVecOpts struct {
		VectorOption
		Buckets     []float64
		ConstLabels map[string]string
	}
	promHistogram struct {
		histogram *prom.HistogramVec
	}
)

func NewHistogram(conf *HistogramVecOpts) Histogram {
	if conf == nil {
		return nil
	}
	vec := prom.NewHistogramVec(prom.HistogramOpts{
		Namespace:   conf.Namespace,
		Subsystem:   conf.Subsystem,
		Name:        conf.Name,
		Help:        conf.Help,
		Buckets:     conf.Buckets,
		ConstLabels: conf.ConstLabels,
	}, conf.Labels)
	prom.MustRegister(vec)
	h := &promHistogram{
		histogram: vec,
	}
	return h
}

// Close implements Histogram.
func (p *promHistogram) Close() error {
	if prom.Unregister(p.histogram) {
		return nil
	}
	return errors.New("failed to unregister histogram metric")
}

// Observe implements Histogram.
func (p *promHistogram) Observe(value float64, labels ...string) {
	update(func() {
		p.histogram.WithLabelValues(labels...).Observe(value)
	})
}

var _ Histogram = (*promHistogram)(nil)
