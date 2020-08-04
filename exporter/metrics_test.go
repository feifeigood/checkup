package exporter

import (
	"reflect"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
)

func TestNewCounterVec(t *testing.T) {
	counter := NewCounterVec(prometheus.CounterOpts{
		Namespace: "checkup",
		Name:      "checks_total",
		Help:      "Total of checks number by checker",
	}, []string{"type", "title", "endpoint"}, 200*time.Millisecond)

	counter.ProcessUpdate([]string{"tcp", "test", "127.0.0.1:80"}, float64(0))

	switch c := counter.Collector().(type) {
	case *prometheus.CounterVec:
		m := io_prometheus_client.Metric{}
		c.WithLabelValues([]string{"tcp", "test", "127.0.0.1:80"}...).Write(&m)
		if *m.Counter.Value != float64(1) {
			t.Errorf("Expected 1 matches, but got %v matches.", *m.Counter.Value)
		}
	default:
		t.Errorf("Unexpected type of metric: %v", reflect.TypeOf(c))
	}
}
