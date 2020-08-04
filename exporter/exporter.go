package exporter

import (
	"time"

	"github.com/feifeigood/checkup/types"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "checkup"

// CheckupExporter expose checkup metrics
type CheckupExporter struct {
	checksAll      CheckupMetric
	checksHealthy  CheckupMetric
	checksDegraded CheckupMetric
	checksDown     CheckupMetric
}

// NewCheckupExporter returns CheckupExporter structure pointer
func NewCheckupExporter() *CheckupExporter {
	c := &CheckupExporter{}

	labels := []string{"type", "title", "endpoint"}
	retention := time.Duration(5 * time.Minute)

	c.checksAll = NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "checks_total",
		Help:      "Total of checks number by checker",
	}, labels, retention)
	prometheus.MustRegister(c.checksAll.Collector())

	c.checksDegraded = NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "checks_degraded",
		Help:      "Total of degraded checks number by checker",
	}, labels, retention)
	prometheus.MustRegister(c.checksDegraded.Collector())

	c.checksHealthy = NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "checks_healthy",
		Help:      "Total of healthy checks number by checker",
	}, labels, retention)
	prometheus.MustRegister(c.checksHealthy.Collector())

	c.checksDown = NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "checks_down",
		Help:      "Total of down checks number by checker",
	}, labels, retention)
	prometheus.MustRegister(c.checksDown.Collector())

	return c
}

// Update convert checkup checker result to metrics
func (c *CheckupExporter) Update(result types.Result) {
	c.checksAll.ProcessUpdate([]string{result.Type, result.Title, result.Endpoint}, float64(0))
	if result.Healthy {
		c.checksHealthy.ProcessUpdate([]string{result.Type, result.Title, result.Endpoint}, float64(0))
	} else if result.Degraded {
		c.checksDegraded.ProcessUpdate([]string{result.Type, result.Title, result.Endpoint}, float64(0))
	} else {
		c.checksDown.ProcessUpdate([]string{result.Type, result.Title, result.Endpoint}, float64(0))
	}
}

// ProcessRetention remove old metric
func (c *CheckupExporter) ProcessRetention() {
	c.checksAll.ProcessRetention()
	c.checksHealthy.ProcessRetention()
	c.checksDegraded.ProcessRetention()
	c.checksDown.ProcessRetention()
}
