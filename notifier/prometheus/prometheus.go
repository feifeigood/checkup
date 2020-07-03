package prometheus

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/feifeigood/checkup/types"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Type should match package name
const Type = "prometheus"

const namespace = "checkup"

var log = logrus.WithField("component", "notifier")

var (
	checksAll = prom.NewCounterVec(prom.CounterOpts{
		Namespace: namespace,
		Name:      "checks_total",
		Help:      "Total of checks number by checker",
	}, []string{"type", "title", "endpoint"})

	checksHealthy = prom.NewCounterVec(prom.CounterOpts{
		Namespace: namespace,
		Name:      "checks_healthy",
		Help:      "Total of healthy checks number by checker",
	}, []string{"type", "title", "endpoint"})

	checksDegraded = prom.NewCounterVec(prom.CounterOpts{
		Namespace: namespace,
		Name:      "checks_degraded",
		Help:      "Total of degraded checks number by checker",
	}, []string{"type", "title", "endpoint"})

	checksDown = prom.NewCounterVec(prom.CounterOpts{
		Namespace: namespace,
		Name:      "checks_down",
		Help:      "Total of down checks number by checker",
	}, []string{"type", "title", "endpoint"})
)

func init() {
	prom.MustRegister(checksAll)
	prom.MustRegister(checksHealthy)
	prom.MustRegister(checksDegraded)
	prom.MustRegister(checksDown)
}

// Notifier is a way for notify health check to prometheus
type Notifier struct {
	Listen      string `json:"listen,omitempty"`
	MetricsPath string `json:"metrics_path,omitempty"`
	BasicAuth   bool   `json:"basic_auth,omitempty"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
}

// New creates a new Notifier based on json config and running http server
func New(config json.RawMessage) (Notifier, error) {
	var notifier Notifier
	err := json.Unmarshal(config, &notifier)

	log.Infof("Running prometheus metrics handler listen: %s, metrics-path: %s", notifier.Listen, notifier.MetricsPath)
	go func() {
		if notifier.BasicAuth {
			http.Handle(notifier.MetricsPath, notifier.auth(promhttp.Handler()))
		} else {
			http.Handle(notifier.MetricsPath, promhttp.Handler())
		}
		log.Fatal(http.ListenAndServe(notifier.Listen, nil))
	}()

	return notifier, err
}

// Type returns the notifier package name
func (Notifier) Type() string {
	return Type
}

// Notify convert health check results to prometheus metrics
func (p Notifier) Notify(results []types.Result) error {
	for _, result := range results {
		checksAll.WithLabelValues(result.Type, result.Title, result.Endpoint).Inc()
		if result.Healthy {
			checksHealthy.WithLabelValues(result.Type, result.Title, result.Endpoint).Inc()
		} else if result.Degraded {
			checksDegraded.WithLabelValues(result.Type, result.Title, result.Endpoint).Inc()
		} else {
			checksDown.WithLabelValues(result.Type, result.Title, result.Endpoint).Inc()
		}
	}
	return nil
}

func (p Notifier) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2)

		if len(auth) != 2 || auth[0] != "Basic" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "authorization failed", http.StatusUnauthorized)
			return
		}

		payload, _ := base64.StdEncoding.DecodeString(auth[1])
		pair := strings.SplitN(string(payload), ":", 2)

		if len(pair) != 2 || pair[0] != p.Username || pair[1] != p.Password {
			http.Error(w, "authorization failed", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
