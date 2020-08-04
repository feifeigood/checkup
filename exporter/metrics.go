package exporter

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// CheckupMetric checkup collector metric interface
type CheckupMetric interface {
	Name() string
	Collector() prometheus.Collector

	ProcessUpdate(lvs []string, v float64)

	// Remove old metrics
	ProcessRetention() error
}

// prometheus metricVec implement this interface
type deleteVec interface {
	Delete(labels prometheus.Labels) bool
}

type metric struct {
	name      string
	retention time.Duration
	tracker   LabelValueTracker
}

func (m *metric) Name() string {
	return m.name
}

func (m *metric) processRetention(vec deleteVec) error {
	if m.retention != 0 {
		for _, lvs := range m.tracker.DeleteByRetention(m.retention) {
			vec.Delete(lvs)
		}
	}
	return nil
}

type counterVec struct {
	metric
	vec *prometheus.CounterVec
}

func (m *counterVec) Collector() prometheus.Collector {
	return m.vec
}

func (m *counterVec) ProcessUpdate(lvs []string, v float64) {
	m.vec.WithLabelValues(lvs...).Inc()
}

func (m *counterVec) ProcessRetention() error {
	return m.processRetention(m.vec)
}

// NewCounterVec returns an CounterVec Metrics
func NewCounterVec(opts prometheus.CounterOpts, labels []string, retention time.Duration) CheckupMetric {
	return &counterVec{
		metric: metric{
			name:      opts.Name,
			retention: retention,
			tracker:   NewLabelValueTracker(labels),
		},
		vec: prometheus.NewCounterVec(opts, labels),
	}
}
