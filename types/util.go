package types

import (
	"fmt"
	"strings"
	"time"
)

// Timestamp returns the UTC Unix timestamp in nanoseconds.
func Timestamp() int64 {
	return time.Now().UTC().UnixNano()
}

// Duration is wrapped time.Duration for JSON
type Duration struct {
	time.Duration
}

// UnmarshalJSON unmarshal string to time.Duration
func (d *Duration) UnmarshalJSON(b []byte) (err error) {
	d.Duration, err = time.ParseDuration(strings.Trim(string(b), `"`))
	return
}

// MarshalJSON marshal time.Duration to string
func (d Duration) MarshalJSON() (b []byte, err error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}
