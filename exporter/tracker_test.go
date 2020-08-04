package exporter

import (
	"testing"
	"time"
)

func TestDeleteByRetention(t *testing.T) {
	tracker := NewLabelValueTracker([]string{"service", "user", "hostname", "country"})
	for _, labels := range []map[string]string{
		{
			"service":  "service a",
			"user":     "alice",
			"hostname": "localhost",
			"country":  "Finland",
		},
		{
			"service":  "service a",
			"user":     "alice",
			"hostname": "localhost",
			"country":  "Norway",
		},
		{
			"service":  "service a",
			"user":     "alice",
			"hostname": "localhost",
			"country":  "Sweden",
		},
	} {
		tracker.Observe(labels)
	}
	time.Sleep(500 * time.Millisecond)
	tracker.Observe(map[string]string{ // already known, should update the timestamp but not create a new entry
		"service":  "service a",
		"user":     "alice",
		"hostname": "localhost",
		"country":  "Norway",
	})
	verify(t, nil, 0, tracker, 3, nil)
	deleted := tracker.DeleteByRetention(250 * time.Millisecond) // remove all but the updated entry
	verify(t, deleted, 2, tracker, 1, nil)
	deleted = tracker.DeleteByRetention(250 * time.Millisecond) // should do nothing, because the remaining entry is newer
	verify(t, deleted, 0, tracker, 1, nil)
}

func verify(t *testing.T, deleted []map[string]string, nDeleted int, tracker LabelValueTracker, nRemaining int, err error) {
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	if len(deleted) != nDeleted {
		t.Fatalf("expected %v deleted entries, but got %v", nDeleted, len(deleted))
	}
	if nEntries(t, tracker) != nRemaining {
		t.Fatalf("expected %v remaining entries, but got %v", nRemaining, nEntries(t, tracker))
	}
}

func nEntries(t *testing.T, tracker LabelValueTracker) int {
	trackerInternal, ok := tracker.(*observedLabels)
	if !ok {
		t.Fatal("Cannot cast tracker to *observedLabelValues")
		return 0
	}
	return len(trackerInternal.values)
}
