package icmp

import (
	"encoding/json"
	"fmt"
	"time"

	checkup_prometheus_client "github.com/feifeigood/checkup/prometheus"
	"github.com/feifeigood/checkup/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Type should match the package name
const Type = "icmp"

// DefaultPacketCount default send icmp packet
const (
	DefaultPacketCount = 10
	DefaultTimeout     = 10 * time.Second
	DefaultInterval    = 500 * time.Millisecond
)

var (
	log        = logrus.WithField("component", "icmp")
	healthy    *prometheus.GaugeVec
	packetLoss *prometheus.GaugeVec
)

func init() {

	healthy = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "checkup_icmp_healthy",
		Help: "Using icmp checker to checks endpoint is healthy",
	}, []string{"title", "endpoint"})

	packetLoss = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "checkup_icmp_packet_loss",
		Help: "Using icmp checker checks the percentage of packets lost",
	}, []string{"title", "endpoint"})

	prometheus.MustRegister(healthy)
	prometheus.MustRegister(packetLoss)
}

// Checker implements a Checker for ICMP endpoints.
type Checker struct {
	// Name is the name of the endpoint.
	Name string `json:"endpoint_name"`

	// URL is the URL of the endpoint.
	URL string `json:"endpoint_url"`

	// Privileged choose protocol is ip or udp
	Privileged bool `json:"privileged,omitempty"`

	// Timeout is the maximum time to wait for a
	// ICMP EchoReply message
	Timeout types.Duration `json:"timeout,omitempty"`

	// Interval is the wait time between each packet send.
	// Default is 1s.
	Interval types.Duration `json:"interval,omitempty"`

	// Count tells pinger to stop after sending (and receiving) Count echo
	// packets. If zero using default value 5
	Count int `json:"count,omitempty"`

	// Every
	Every types.Duration `json:"every,omitempty"`

	// ThresholdRTT is the maximum round trip time to
	// allow for a healthy endpoint. If non-zero and a
	// request takes longer than ThresholdRTT, the
	// endpoint will be considered unhealthy. Note that
	// this duration includes any in-between network
	// latency.
	ThresholdRTT time.Duration `json:"threshold_rtt,omitempty"`

	// Attempts is how many requests the client will
	// make to the endpoint in a single check.
	Attempts int `json:"attempts,omitempty"`

	stats *Statistics
}

// New creates a new Checker instance based on json config
func New(config json.RawMessage) (*Checker, error) {
	var checker Checker
	err := json.Unmarshal(config, &checker)
	return &checker, err
}

// Type returns the checker package name
func (c *Checker) Type() string {
	return Type
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

	if c.Count < 1 {
		c.Count = DefaultPacketCount
	}

	if c.Timeout.Duration < 1 {
		c.Timeout.Duration = DefaultTimeout
	}

	if c.Interval.Duration < 1 {
		c.Interval.Duration = DefaultInterval
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
	checks := make(types.Attempts, c.Attempts)

	pinger, _ := NewPinger(c.URL)
	pinger.SetPrivileged(c.Privileged)
	pinger.Count = c.Count
	pinger.Timeout = c.Timeout.Duration
	pinger.Interval = c.Interval.Duration

	for i := 0; i < c.Attempts; i++ {
		start := time.Now()

		err := pinger.Run()

		checks[i].RTT = time.Since(start)
		if err != nil {
			checks[i].Error = err.Error()
			continue
		}
		c.stats = pinger.Statistics()

		err = c.checkDown(c.stats)
		if err != nil {
			checks[i].Error = err.Error()
		}
	}

	return checks
}

func (c *Checker) checkDown(stats *Statistics) error {

	if stats.PacketLoss == 100 {
		return fmt.Errorf("didn't receive any icmp packets")
	}

	return nil
}

// conclude takes the data in result from the attempts and
// computes remaining values needed to fill out the result.
// It detects degraded (high-latency) responses and makes
// the conclusion about the result's status.
func (c *Checker) conclude(result types.Result) types.Result {
	result.ThresholdRTT = c.ThresholdRTT

	if c.stats == nil {
		healthy.WithLabelValues(result.Title, result.Endpoint).Set(float64(0))
	} else {
		healthy.WithLabelValues(result.Title, result.Endpoint).Set(float64(1))
		packetLoss.WithLabelValues(result.Title, result.Endpoint).Set(c.stats.PacketLoss)
	}
	c.stats = nil

	// Check errors (down)
	for i := range result.Times {
		if result.Times[i].Error != "" {
			result.Down = true
			return result
		}
	}

	// Check round trip time (degraded)
	if c.ThresholdRTT > 0 {
		stats := result.ComputeStats()
		if stats.Median > c.ThresholdRTT {
			result.Notice = fmt.Sprintf("median round trip time exceeded threshold (%s)", c.ThresholdRTT)
			result.Degraded = true
			return result
		}
	}

	result.Healthy = true
	return result
}

func (c *Checker) Collect(collector checkup_prometheus_client.Collector) {

}
