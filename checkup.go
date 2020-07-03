package checkup

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/feifeigood/checkup/check/http"
	"github.com/feifeigood/checkup/check/tcp"
	"github.com/feifeigood/checkup/notifier/prometheus"
	"github.com/feifeigood/checkup/storage/fs"
	"github.com/feifeigood/checkup/types"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("component", "checkup")

// DefaultConcurrentChecks is how many checks, at most to perform concurrently
var DefaultConcurrentChecks = 128

// Checker can create a types.Result.
type Checker interface {
	Type() string
	Check() (types.Result, error)
}

// Notifier can notify a types.Result.
type Notifier interface {
	Type() string
	Notify([]types.Result) error
}

// Storage can store results.
type Storage interface {
	Type() string
	Store([]types.Result) error
}

// StorageReader can read results from the Storage.
type StorageReader interface {
	// Fetch returns the contents of a check file.
	Fetch(checkFile string) ([]types.Result, error)
	// GetIndex returns the storage index, as a map where keys are check
	// result filenames and values are the associated check timestamps.
	GetIndex() (map[string]int64, error)
}

// Maintainer can maintain a store of results by
// deleting old check files that are no longer
// needed or performing other required tasks.
type Maintainer interface {
	Maintain() error
}

const (
	errUnknownCheckerType  = "unknown checker type: %s"
	errUnknownStorageType  = "unknown storage type: %s"
	errUnknownNotifierType = "unknown notifier type: %s"
)

// Checkup performs a routine checkup on endpoints or services.
type Checkup struct {
	// Checkers is the list of Checkers of use with which to perform checks.
	Checkers []Checker `json:"checkers,omitempty"`

	// ConcurrentChecks is how many checks, at most, to perform concurrently.
	// Default value is DefaultConcurrentChecks
	ConcurrentChecks int `json:"concurrent_checks,omitempty"`

	// Storage is the storage mechanism for saving the
	// results of checks.
	Storage Storage `json:"storage,omitempty"`

	// Notifiers
	Notifiers []Notifier `json:"notifiers,omitempty"`
}

// Check perform the health checks.
func (c Checkup) Check() ([]types.Result, error) {
	if c.ConcurrentChecks == 0 {
		c.ConcurrentChecks = DefaultConcurrentChecks
	}

	if c.ConcurrentChecks < 0 {
		return nil, fmt.Errorf("invalid value for Concurrentchecks: %d (must be set > 0", c.ConcurrentChecks)
	}

	results := make([]types.Result, len(c.Checkers))
	errs := make(types.Errors, len(c.Checkers))
	throttle := make(chan struct{}, c.ConcurrentChecks)
	wg := sync.WaitGroup{}

	for i, checker := range c.Checkers {
		throttle <- struct{}{}
		wg.Add(1)
		go func(i int, checker Checker) {
			results[i], errs[i] = checker.Check()
			log.Debugf("== (%s)%s - %s - %s", results[i].Type, results[i].Title, results[i].Endpoint, results[i].Status())
			<-throttle
			wg.Done()
		}(i, checker)
	}
	wg.Wait()

	if !errs.Empty() {
		return nil, errs
	}

	for _, service := range c.Notifiers {
		err := service.Notify(results)
		if err != nil {
			log.Errorf("sending notifications for %s: %s", service.Type(), err)
		}
	}

	return results, nil
}

// CheckAndStore performs health checks and immediately
// stores the results to the configured storage if there
// were no errors.
func (c Checkup) CheckAndStore() error {
	if c.Storage == nil {
		return fmt.Errorf("no storage mechanism defined")
	}

	results, err := c.Check()
	if err != nil {
		return err
	}

	err = c.Storage.Store(results)
	if err != nil {
		return err
	}

	if m, ok := c.Storage.(Maintainer); ok {
		return m.Maintain()
	}

	return nil
}

// CheckAndStoreEvery calls CheckAndStore every interval.
func (c Checkup) CheckAndStoreEvery(interval time.Duration) *time.Ticker {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := c.CheckAndStore(); err != nil {
				log.Error(err)
			}
		}
	}()
	return ticker
}

// UnmarshalJSON unmarshales b into c. To succeed, it
// requires type information for the interface values.
func (c *Checkup) UnmarshalJSON(b []byte) error {

	type checkup2 *Checkup
	_ = json.Unmarshal(b, checkup2(c))

	// clean the slate
	c.Checkers = []Checker{}
	c.Notifiers = []Notifier{}

	// Begin unmarshaling interface values by
	// collecting the raw JSON
	raw := struct {
		Checkers  []json.RawMessage `json:"checkers"`
		Storage   json.RawMessage   `json:"storage"`
		Notifiers []json.RawMessage `json:"notifiers"`
	}{}

	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}

	// Then collect the concrete type information
	configTypes := struct {
		Checkers []struct {
			Type string `json:"type"`
		}
		Storage struct {
			Type string `json:"type"`
		}
		Notifiers []struct {
			Type string `json:"type"`
		}
	}{}

	err = json.Unmarshal(b, &configTypes)
	if err != nil {
		return err
	}

	for i, t := range configTypes.Checkers {
		checker, err := checkerDecode(t.Type, raw.Checkers[i])
		if err != nil {
			return err
		}
		c.Checkers = append(c.Checkers, checker)
	}

	if raw.Storage != nil {
		storage, err := storageDecode(configTypes.Storage.Type, raw.Storage)
		if err != nil {
			return err
		}
		c.Storage = storage
	}

	for i, n := range configTypes.Notifiers {
		notifier, err := notifierDecode(n.Type, raw.Notifiers[i])
		if err != nil {
			return err
		}
		c.Notifiers = append(c.Notifiers, notifier)
	}

	return nil
}

func checkerDecode(typeName string, config json.RawMessage) (Checker, error) {
	switch typeName {
	// case dns.Type:
	// 	return dns.New(config)
	// case exec.Type:
	// 	return exec.New(config)
	case http.Type:
		return http.New(config)
	case tcp.Type:
		return tcp.New(config)
	// case tls.Type:
	// 	return tls.New(config)
	default:
		return nil, fmt.Errorf(errUnknownCheckerType, typeName)
	}
}

func storageDecode(typeName string, config json.RawMessage) (Storage, error) {
	switch typeName {
	// case s3.Type:
	// 	return s3.New(config)
	// case github.Type:
	// 	return github.New(config)
	case fs.Type:
		return fs.New(config)
	// case sql.Type:
	// 	return sql.New(config)
	default:
		return nil, fmt.Errorf(errUnknownStorageType, typeName)
	}
}

func notifierDecode(typeName string, config json.RawMessage) (Notifier, error) {
	switch typeName {
	case prometheus.Type:
		return prometheus.New(config)
	// case mail.Type:
	// 	return mail.New(config)
	// case slack.Type:
	// 	return slack.New(config)
	// case mailgun.Type:
	// 	return mailgun.New(config)
	// case pushover.Type:
	// 	return pushover.New(config)
	default:
		return nil, fmt.Errorf(errUnknownNotifierType, typeName)
	}
}
