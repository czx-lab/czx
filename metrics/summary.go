package metrics

import (
	"errors"

	prom "github.com/prometheus/client_golang/prometheus"
)

type (
	// A SummaryVecOpts is a summary vector options
	SummaryVecOpts struct {
		VecOpt     VectorOption
		Objectives map[float64]float64
	}

	promSummary struct {
		summary *prom.SummaryVec
	}
)

var _ Summary = (*promSummary)(nil)

func NewSummary(conf *SummaryVecOpts) Summary {
	if conf == nil {
		return nil
	}
	vec := prom.NewSummaryVec(prom.SummaryOpts{
		Namespace:  conf.VecOpt.Namespace,
		Subsystem:  conf.VecOpt.Subsystem,
		Name:       conf.VecOpt.Name,
		Help:       conf.VecOpt.Help,
		Objectives: conf.Objectives,
	}, conf.VecOpt.Labels)
	prom.MustRegister(vec)
	s := &promSummary{
		summary: vec,
	}
	return s
}

// Close implements Summary.
func (p *promSummary) Close() error {
	if prom.Unregister(p.summary) {
		return nil
	}
	return errors.New("failed to unregister summary metric")
}

// Observe implements Summary.
func (p *promSummary) Observe(value float64, labels ...string) {
	update(func() {
		p.summary.WithLabelValues(labels...).Observe(value)
	})
}
