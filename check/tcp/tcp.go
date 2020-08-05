package tcp

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	checkup_prometheus_client "github.com/feifeigood/checkup/prometheus"
	"github.com/feifeigood/checkup/prometheus/metric"
	"github.com/feifeigood/checkup/types"
)

var (
	errReadingRootCert = errors.New("error reading root certificate")
	errParsingRootCert = errors.New("error parsing root certificate")
)

// Type should match the package name
const Type = "tcp"

// Checker implements a Checker for TCP endpoints.
type Checker struct {
	// Name is the name of the endpoint.
	Name string `json:"endpoint_name"`

	// URL is the URL of the endpoint.
	URL string `json:"endpoint_url"`

	// TLSEnabled controls whether to enable TLS or not.
	// If set, TLS is enabled.
	TLSEnabled bool `json:"tls,omitempty"`

	// TLSSkipVerify controls whether to skip server TLS
	// certificat validation or not.
	TLSSkipVerify bool `json:"tls_skip_verify,omitempty"`

	// TLSCAFile is the Certificate Authority used
	// to validate the server TLS certificate.
	TLSCAFile string `json:"tls_ca_file,omitempty"`

	// Timeout is the maximum time to wait for a
	// TCP connection to be established.
	Timeout types.Duration `json:"timeout,omitempty"`

	// ThresholdRTT is the maximum round trip time to
	// allow for a healthy endpoint. If non-zero and a
	// request takes longer than ThresholdRTT, the
	// endpoint will be considered unhealthy. Note that
	// this duration includes any in-between network
	// latency.
	ThresholdRTT types.Duration `json:"threshold_rtt,omitempty"`

	// Attempts is how many requests the client will
	// make to the endpoint in a single check.
	Attempts int `json:"attempts,omitempty"`

	// Every override every subcommand interval If set.
	Every types.Duration `json:"every,omitempty"`

	//
	metrics []metric.Metric
}

// Type returns the checker package name
func (c *Checker) Type() string {
	return Type
}

// New creates a new Checker instance based on json config
func New(config json.RawMessage) (*Checker, error) {
	var checker Checker
	err := json.Unmarshal(config, &checker)
	checker.metrics = []metric.Metric{}
	return &checker, err
}

// GetEvery returns the checker specified check interval to override every subcommand
func (c *Checker) GetEvery() time.Duration {
	return c.Every.Duration
}

// Check performs checks using c according to its configuration.
// An error is only returned if there is a configuration error.
func (c *Checker) Check() (types.Result, error) {
	if c.Attempts < 1 {
		c.Attempts = 1
	}

	result := types.NewResult()
	result.Type = c.Type()
	result.Title = c.Name
	result.Endpoint = c.URL
	result.Times = c.doChecks()

	return c.conclude(result), nil
}

// doChecks executes and returns each attempt.
func (c *Checker) doChecks() types.Attempts {
	var err error
	var conn net.Conn

	timeout := c.Timeout.Duration
	if timeout == 0 {
		timeout = time.Second
	}

	dialer := func() (net.Conn, error) {
		return net.DialTimeout("tcp", c.URL, timeout)
	}
	if c.TLSEnabled {
		dialer = func() (net.Conn, error) {
			// Dialer with timeout
			dialer := &net.Dialer{
				Timeout: timeout,
			}

			// TLS config based on configuration
			var tlsConfig tls.Config
			tlsConfig.InsecureSkipVerify = c.TLSSkipVerify
			if c.TLSCAFile != "" {
				rootPEM, err := ioutil.ReadFile(c.TLSCAFile)
				if err != nil || rootPEM == nil {
					return nil, errReadingRootCert
				}
				pool := x509.NewCertPool()
				ok := pool.AppendCertsFromPEM(rootPEM)
				if !ok {
					return nil, errParsingRootCert
				}
				tlsConfig.RootCAs = pool
			}
			return tls.DialWithDialer(dialer, "tcp", c.URL, &tlsConfig)
		}
	}

	checks := make(types.Attempts, c.Attempts)
	for i := 0; i < c.Attempts; i++ {
		start := time.Now()

		if conn, err = dialer(); err == nil {
			conn.Close()
		}

		checks[i].RTT = time.Since(start)
		if err != nil {
			checks[i].Error = err.Error()
			continue
		}
	}
	return checks
}

// conclude takes the data in result from the attempts and
// computes remaining values needed to fill out the result.
// It detects degraded (high-latency) responses and makes
// the conclusion about the result's status.
func (c *Checker) conclude(result types.Result) types.Result {
	result.ThresholdRTT = c.ThresholdRTT.Duration

	// Check errors (down)
	for i := range result.Times {
		if result.Times[i].Error != "" {
			result.Down = true
			m, _ := metric.New(
				"checkup_tcp",
				map[string]string{
					"title":    result.Title,
					"endpoint": result.Endpoint,
				}, map[string]interface{}{
					"healthy": 0,
				}, time.Now(), metric.Gauge,
			)
			c.metrics = append(c.metrics, m)
			return result
		}
	}

	m, _ := metric.New(
		"checkup_tcp",
		map[string]string{
			"title":    result.Title,
			"endpoint": result.Endpoint,
		}, map[string]interface{}{
			"healthy": 1,
		}, time.Now(), metric.Gauge,
	)
	c.metrics = append(c.metrics, m)

	// Check round trip time (degraded)
	if c.ThresholdRTT.Duration > 0 {
		stats := result.ComputeStats()
		if stats.Median > c.ThresholdRTT.Duration {
			result.Notice = fmt.Sprintf("median round trip time exceeded threshold (%s)", c.ThresholdRTT)
			result.Degraded = true
			return result
		}
	}

	result.Healthy = true
	return result
}

func (c *Checker) Collect(collector checkup_prometheus_client.Collector) {
	if c.metrics != nil && len(c.metrics) > 0 {
		collector.Add(c.metrics)
	}
	c.metrics = []metric.Metric{}
}
