package metric

import (
	"fmt"
	"hash/fnv"
	"sort"
	"time"
)

// ValueType is an enumeration of metric types that represent a simple value.
type ValueType int

// Possible values for the ValueType enum.
const (
	_ ValueType = iota
	Counter
	Gauge
	Untyped
	Summary
	Histogram
)

// Tag represents a single tag key and value.
type Tag struct {
	Key   string
	Value string
}

// Field represents a single field key and value.
type Field struct {
	Key   string
	Value interface{}
}

// Metric is the type of data that is processed by Telegraf.  Input plugins,
// and to a lesser degree, Processor and Aggregator plugins create new Metrics
// and Output plugins write them.
type Metric interface {
	// Name is the primary identifier for the Metric and corresponds to the
	// measurement in the InfluxDB data model.
	Name() string

	// Tags returns the tags as a map.  This method is deprecated, use TagList instead.
	Tags() map[string]string

	// TagList returns the tags as a slice ordered by the tag key in lexical
	// bytewise ascending order.  The returned value should not be modified,
	// use the AddTag or RemoveTag methods instead.
	TagList() []*Tag

	// Fields returns the fields as a map.  This method is deprecated, use FieldList instead.
	Fields() map[string]interface{}

	// FieldList returns the fields as a slice in an undefined order.  The
	// returned value should not be modified, use the AddField or RemoveField
	// methods instead.
	FieldList() []*Field

	// Time returns the timestamp of the metric.
	Time() time.Time

	// Type returns a general type for the entire metric that describes how you
	// might interpret, aggregate the values.
	//
	// This method may be removed in the future and its use is discouraged.
	Type() ValueType

	// SetName sets the metric name.
	SetName(name string)

	// AddPrefix adds a string to the front of the metric name.  It is
	// equivalent to m.SetName(prefix + m.Name()).
	//
	// This method is deprecated, use SetName instead.
	AddPrefix(prefix string)

	// AddSuffix appends a string to the back of the metric name.  It is
	// equivalent to m.SetName(m.Name() + suffix).
	//
	// This method is deprecated, use SetName instead.
	AddSuffix(suffix string)

	// GetTag returns the value of a tag and a boolean to indicate if it was set.
	GetTag(key string) (string, bool)

	// HasTag returns true if the tag is set on the Metric.
	HasTag(key string) bool

	// AddTag sets the tag on the Metric.  If the Metric already has the tag
	// set then the current value is replaced.
	AddTag(key, value string)

	// RemoveTag removes the tag if it is set.
	RemoveTag(key string)

	// GetField returns the value of a field and a boolean to indicate if it was set.
	GetField(key string) (interface{}, bool)

	// HasField returns true if the field is set on the Metric.
	HasField(key string) bool

	// AddField sets the field on the Metric.  If the Metric already has the field
	// set then the current value is replaced.
	AddField(key string, value interface{})

	// RemoveField removes the tag if it is set.
	RemoveField(key string)

	// SetTime sets the timestamp of the Metric.
	SetTime(t time.Time)

	// HashID returns an unique identifier for the series.
	HashID() uint64

	// Copy returns a deep copy of the Metric.
	Copy() Metric

	// Accept marks the metric as processed successfully and written to an
	// output.
	Accept()

	// Reject marks the metric as processed unsuccessfully.
	Reject()

	// Drop marks the metric as processed successfully without being written
	// to any output.
	Drop()

	// SetAggregate indicates the metric is an aggregated value.
	//
	// This method may be removed in the future and its use is discouraged.
	SetAggregate(bool)

	// IsAggregate returns true if the Metric is an aggregate.
	//
	// This method may be removed in the future and its use is discouraged.
	IsAggregate() bool
}

type metric struct {
	name   string
	tags   []*Tag
	fields []*Field
	tm     time.Time

	tp        ValueType
	aggregate bool
}

func New(
	name string,
	tags map[string]string,
	fields map[string]interface{},
	tm time.Time,
	tp ...ValueType,
) (Metric, error) {
	var vtype ValueType
	if len(tp) > 0 {
		vtype = tp[0]
	} else {
		vtype = Untyped
	}

	m := &metric{
		name:   name,
		tags:   nil,
		fields: nil,
		tm:     tm,
		tp:     vtype,
	}

	if len(tags) > 0 {
		m.tags = make([]*Tag, 0, len(tags))
		for k, v := range tags {
			m.tags = append(m.tags,
				&Tag{Key: k, Value: v})
		}
		sort.Slice(m.tags, func(i, j int) bool { return m.tags[i].Key < m.tags[j].Key })
	}

	if len(fields) > 0 {
		m.fields = make([]*Field, 0, len(fields))
		for k, v := range fields {
			v := convertField(v)
			if v == nil {
				continue
			}
			m.AddField(k, v)
		}
	}

	return m, nil
}

// FromMetric returns a deep copy of the metric with any tracking information
// removed.
func FromMetric(other Metric) Metric {
	m := &metric{
		name:      other.Name(),
		tags:      make([]*Tag, len(other.TagList())),
		fields:    make([]*Field, len(other.FieldList())),
		tm:        other.Time(),
		tp:        other.Type(),
		aggregate: other.IsAggregate(),
	}

	for i, tag := range other.TagList() {
		m.tags[i] = &Tag{Key: tag.Key, Value: tag.Value}
	}

	for i, field := range other.FieldList() {
		m.fields[i] = &Field{Key: field.Key, Value: field.Value}
	}
	return m
}

func (m *metric) String() string {
	return fmt.Sprintf("%s %v %v %d", m.name, m.Tags(), m.Fields(), m.tm.UnixNano())
}

func (m *metric) Name() string {
	return m.name
}

func (m *metric) Tags() map[string]string {
	tags := make(map[string]string, len(m.tags))
	for _, tag := range m.tags {
		tags[tag.Key] = tag.Value
	}
	return tags
}

func (m *metric) TagList() []*Tag {
	return m.tags
}

func (m *metric) Fields() map[string]interface{} {
	fields := make(map[string]interface{}, len(m.fields))
	for _, field := range m.fields {
		fields[field.Key] = field.Value
	}

	return fields
}

func (m *metric) FieldList() []*Field {
	return m.fields
}

func (m *metric) Time() time.Time {
	return m.tm
}

func (m *metric) Type() ValueType {
	return m.tp
}

func (m *metric) SetName(name string) {
	m.name = name
}

func (m *metric) AddPrefix(prefix string) {
	m.name = prefix + m.name
}

func (m *metric) AddSuffix(suffix string) {
	m.name = m.name + suffix
}

func (m *metric) AddTag(key, value string) {
	for i, tag := range m.tags {
		if key > tag.Key {
			continue
		}

		if key == tag.Key {
			tag.Value = value
			return
		}

		m.tags = append(m.tags, nil)
		copy(m.tags[i+1:], m.tags[i:])
		m.tags[i] = &Tag{Key: key, Value: value}
		return
	}

	m.tags = append(m.tags, &Tag{Key: key, Value: value})
}

func (m *metric) HasTag(key string) bool {
	for _, tag := range m.tags {
		if tag.Key == key {
			return true
		}
	}
	return false
}

func (m *metric) GetTag(key string) (string, bool) {
	for _, tag := range m.tags {
		if tag.Key == key {
			return tag.Value, true
		}
	}
	return "", false
}

func (m *metric) RemoveTag(key string) {
	for i, tag := range m.tags {
		if tag.Key == key {
			copy(m.tags[i:], m.tags[i+1:])
			m.tags[len(m.tags)-1] = nil
			m.tags = m.tags[:len(m.tags)-1]
			return
		}
	}
}

func (m *metric) AddField(key string, value interface{}) {
	for i, field := range m.fields {
		if key == field.Key {
			m.fields[i] = &Field{Key: key, Value: convertField(value)}
			return
		}
	}
	m.fields = append(m.fields, &Field{Key: key, Value: convertField(value)})
}

func (m *metric) HasField(key string) bool {
	for _, field := range m.fields {
		if field.Key == key {
			return true
		}
	}
	return false
}

func (m *metric) GetField(key string) (interface{}, bool) {
	for _, field := range m.fields {
		if field.Key == key {
			return field.Value, true
		}
	}
	return nil, false
}

func (m *metric) RemoveField(key string) {
	for i, field := range m.fields {
		if field.Key == key {
			copy(m.fields[i:], m.fields[i+1:])
			m.fields[len(m.fields)-1] = nil
			m.fields = m.fields[:len(m.fields)-1]
			return
		}
	}
}

func (m *metric) SetTime(t time.Time) {
	m.tm = t
}

func (m *metric) Copy() Metric {
	m2 := &metric{
		name:      m.name,
		tags:      make([]*Tag, len(m.tags)),
		fields:    make([]*Field, len(m.fields)),
		tm:        m.tm,
		tp:        m.tp,
		aggregate: m.aggregate,
	}

	for i, tag := range m.tags {
		m2.tags[i] = &Tag{Key: tag.Key, Value: tag.Value}
	}

	for i, field := range m.fields {
		m2.fields[i] = &Field{Key: field.Key, Value: field.Value}
	}
	return m2
}

func (m *metric) SetAggregate(b bool) {
	m.aggregate = true
}

func (m *metric) IsAggregate() bool {
	return m.aggregate
}

func (m *metric) HashID() uint64 {
	h := fnv.New64a()
	h.Write([]byte(m.name))
	h.Write([]byte("\n"))
	for _, tag := range m.tags {
		h.Write([]byte(tag.Key))
		h.Write([]byte("\n"))
		h.Write([]byte(tag.Value))
		h.Write([]byte("\n"))
	}
	return h.Sum64()
}

func (m *metric) Accept() {
}

func (m *metric) Reject() {
}

func (m *metric) Drop() {
}

// Convert field to a supported type or nil if unconvertible
func convertField(v interface{}) interface{} {
	switch v := v.(type) {
	case float64:
		return v
	case int64:
		return v
	case string:
		return v
	case bool:
		return v
	case int:
		return int64(v)
	case uint:
		return uint64(v)
	case uint64:
		return uint64(v)
	case []byte:
		return string(v)
	case int32:
		return int64(v)
	case int16:
		return int64(v)
	case int8:
		return int64(v)
	case uint32:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint8:
		return uint64(v)
	case float32:
		return float64(v)
	case *float64:
		if v != nil {
			return *v
		}
	case *int64:
		if v != nil {
			return *v
		}
	case *string:
		if v != nil {
			return *v
		}
	case *bool:
		if v != nil {
			return *v
		}
	case *int:
		if v != nil {
			return int64(*v)
		}
	case *uint:
		if v != nil {
			return uint64(*v)
		}
	case *uint64:
		if v != nil {
			return uint64(*v)
		}
	case *[]byte:
		if v != nil {
			return string(*v)
		}
	case *int32:
		if v != nil {
			return int64(*v)
		}
	case *int16:
		if v != nil {
			return int64(*v)
		}
	case *int8:
		if v != nil {
			return int64(*v)
		}
	case *uint32:
		if v != nil {
			return uint64(*v)
		}
	case *uint16:
		if v != nil {
			return uint64(*v)
		}
	case *uint8:
		if v != nil {
			return uint64(*v)
		}
	case *float32:
		if v != nil {
			return float64(*v)
		}
	default:
		return nil
	}
	return nil
}
