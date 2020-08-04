package exporter

import (
	"fmt"
	"time"
)

// LabelValueTracker track of labels values for prometheus metrics
type LabelValueTracker interface {
	Observe(labels map[string]string) (bool, error)
	DeleteByRetention(retention time.Duration) []map[string]string
}

// Represents the label values for a single time series, i.e. if a time series was created with
//     myVec.WithLabelValues("404", "GET").Add(42)
// then a labelValues with values = []{"404", "GET"} and the current timestamp is created.
type observedLabelValues struct {
	values     []string
	lastUpdate time.Time
}

// Represents a list of labels for all time series ever observed (unless they are deleted).
type observedLabels struct {
	labelNames []string
	values     []*observedLabelValues
}

// NewLabelValueTracker returns an metrics labels values tracker
func NewLabelValueTracker(labelsNames []string) LabelValueTracker {
	return &observedLabels{
		labelNames: labelsNames,
		values:     make([]*observedLabelValues, 0),
	}
}

func (observed *observedLabels) Observe(labels map[string]string) (bool, error) {
	for _, err := range []error{
		observed.assertLabelNamesExist(labels),
		observed.assertLabelNamesComplete(labels),
		observed.assertLabelValuesNotEmpty(labels),
	} {
		if err != nil {
			return false, fmt.Errorf("error observing label values: %v", err)
		}
	}

	values := observed.makeLabelValues(labels)
	return observed.addOrUpdate(values), nil
}

func (observed *observedLabels) assertLabelNamesExist(labels map[string]string) error {
	for key := range labels {
		if !containsString(observed.labelNames, key) {
			return fmt.Errorf("label '%v' is not defined for the metric", key)
		}
	}
	return nil
}

func (observed *observedLabels) DeleteByRetention(retention time.Duration) []map[string]string {
	retentionTime := time.Now().Add(-retention)
	deleted := make([]map[string]string, 0)
	remaining := make([]*observedLabelValues, 0, len(observed.values))
	for _, observedValues := range observed.values {
		if observedValues.lastUpdate.Before(retentionTime) {
			deleted = append(deleted, observed.values2map(observedValues))
		} else {
			remaining = append(remaining, observedValues)
		}
	}
	observed.values = remaining
	return deleted
}

func (observed *observedLabels) values2map(observedValues *observedLabelValues) map[string]string {
	result := make(map[string]string)
	for i := range observedValues.values {
		result[observed.labelNames[i]] = observedValues.values[i]
	}
	return result
}

func (observed *observedLabels) assertLabelNamesComplete(labels map[string]string) error {
	if len(observed.labelNames) != len(labels) {
		return fmt.Errorf("got %v label(s), but the metric was initialized with %v label(s) %v", len(labels), len(observed.labelNames), observed.labelNames)
	}
	return nil
}

// If we want to support empty label values, we must refactor DeleteByLabels(),
// because currently empty label values represent wildcards for deleting.
func (observed *observedLabels) assertLabelValuesNotEmpty(labels map[string]string) error {
	for name, val := range labels {
		if len(val) == 0 {
			return fmt.Errorf("label %v is empty. empty values are not supported", name)
		}
	}
	return nil
}

func (observed *observedLabels) makeLabelValues(labels map[string]string) []string {
	result := make([]string, len(observed.labelNames))
	for i, name := range observed.labelNames {
		result[i] = labels[name] // Missing labels are represented as empty strings.
	}
	return result
}

func (observed *observedLabels) addOrUpdate(values []string) bool {
	for _, observedValues := range observed.values {
		if equals(values, observedValues.values) {
			observedValues.lastUpdate = time.Now()
			return false
		}
	}
	observed.values = append(observed.values, &observedLabelValues{
		values:     values,
		lastUpdate: time.Now(),
	})
	return true
}

// test if the string 's' is contained in 'l'
func containsString(l []string, s string) bool {
	for i := range l {
		if l[i] == s {
			return true
		}
	}
	return false
}

func equals(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// test if the strings in 'a' are the same as the strings in 'b', but treat empty strings as a wildcard
func equalsIgnoreEmpty(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) > 0 && len(b[i]) > 0 && a[i] != b[i] {
			return false
		}
	}
	return true
}
