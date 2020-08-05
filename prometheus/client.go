package prometheus

import (
	checkup_metric "github.com/feifeigood/checkup/prometheus/metric"
	"github.com/prometheus/client_golang/prometheus"
)

type Collector interface {
	Describe(ch chan<- *prometheus.Desc)
	Collect(ch chan<- prometheus.Metric)
	Add(metrics []checkup_metric.Metric) error
}
