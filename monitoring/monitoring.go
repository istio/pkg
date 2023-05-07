// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package monitoring

import (
	"sync"
)

// OpenCensus independent metrics
// Removed unused code:
// - Metric.Decrement

type (
	// A Metric collects numerical observations.
	Metric interface {
		// Increment records a value of 1 for the current measure. For Sums,
		// this is equivalent to adding 1 to the current value. For Gauges,
		// this is equivalent to setting the value to 1. For Distributions,
		// this is equivalent to making an observation of value 1.
		Increment()

		// Decrement records a value of -1 for the current measure. For Sums,
		// this is equivalent to subtracting -1 to the current value. For Gauges,
		// this is equivalent to setting the value to -1. For Distributions,
		// this is equivalent to making an observation of value -1.
		//
		// Not used in istio, removed pending new interface.
		//Decrement()

		// Name returns the name value of a Metric.
		// TODO: internal use only
		Name() string

		// Record makes an observation of the provided value for the given measure.
		// Majority of Istio is setting this to an int
		Record(value float64)

		// RecordInt makes an observation of the provided value for the measure.
		// Not actually used in Istio.
		//RecordInt(value int64)

		// Prometheus uses ConterVec and With(map[string]string) returning Counter
		//
		// Expvar doesn't support labels - but can be used in the name, creating a new expvar in a map
		// The labels are really made part of the name when exporting and is the name of the TS
		//
		// Otel uses attribute.Int("name", val) and similar - from the stable package -
		// but doesn't create a new Metric, it is an option to Add().
		//
		// Istio only uses string/string - so using the pattern from slog works.
		// Once we adopt slog, we can start using slog.Attr pattern

		// With creates a new Metric, with the LabelValues provided. This allows creating
		// a set of pre-dimensioned data for recording purposes. This is primarily used
		// for documentation and convenience. Metrics created with this method do not need
		// to be registered (they share the registration of their parent Metric).
		With(labelValues ...string) Metric

		// Register configures the Metric for export. It MUST be called before collection
		// of values for the Metric. An error will be returned if registration fails.
		// TODO: internal use only
		Register() error
	}

	// DerivedMetrics can be used to supply values that dynamically derive from internal
	// state, but are not updated based on any specific event. Their value will be calculated
	// based on a value func that executes when the metrics are exported.
	//
	// At the moment, only a Gauge type is supported.
	DerivedMetric interface {
		// Name returns the name value of a DerivedMetric.
		Name() string

		// Register handles any required setup to ensure metric export.
		Register() error

		// ValueFrom is used to update the derived value with the provided
		// function and the associated label values. If the metric is unlabeled,
		// ValueFrom may be called without any labelValues. Otherwise, the labelValues
		// supplied MUST match the label keys supplied at creation time both in number
		// and in order.
		ValueFrom(valueFn func() float64, labelValues ...string)
	}

	disabledMetric struct {
		name string
	}

	// Options encode changes to the options passed to a Metric at creation time.
	Options func(*options)

	// DerivedOptions encode changes to the options passed to a DerivedMetric at creation time.
	DerivedOptions func(*derivedOptions)

	// A Label provides a named dimension for a Metric.
	//Label Key

	options struct {
		unit     Unit
		labels   []string // Label
		useInt64 bool
	}

	derivedOptions struct {
		labelKeys []string
		valueFn   func() float64
	}
)

func (dm *disabledMetric) ValueFrom(valueFn func() float64, labelValues ...string) {
}

// Decrement implements Metric
func (dm *disabledMetric) Decrement() {}

// Increment implements Metric
func (dm *disabledMetric) Increment() {}

// Name implements Metric
func (dm *disabledMetric) Name() string {
	return dm.name
}

// Record implements Metric
func (dm *disabledMetric) Record(value float64) {}

// RecordInt implements Metric
func (dm *disabledMetric) RecordInt(value int64) {}

// Register implements Metric
func (dm *disabledMetric) Register() error {
	return nil
}

// With implements Metric
func (dm *disabledMetric) With(labelValues ...string) Metric {
	return dm
}

var _ Metric = &disabledMetric{}

var (
	recordHookMutex sync.RWMutex
)

// WithLabels provides configuration options for a new Metric, providing the expected
// dimensions for data collection for that Metric.
func WithLabels(labels ...string) Options {
	return func(opts *options) {
		opts.labels = labels
	}
}

// WithUnit provides configuration options for a new Metric, providing unit of measure
// information for a new Metric.
func WithUnit(unit Unit) Options {
	return func(opts *options) {
		opts.unit = unit
	}
}

// WithInt64Values provides configuration options for a new Metric, indicating that
// recorded values will be saved as int64 values. Any float64 values recorded will
// converted to int64s via math.Floor-based conversion.
func WithInt64Values() Options {
	return func(opts *options) {
		opts.useInt64 = true
	}
}

// WithLabelKeys is used to configure the label keys used by a DerivedMetric. This
// option is mutually exclusive with the derived option `WithValueFrom` and will be ignored
// if that option is provided.
func WithLabelKeys(keys ...string) DerivedOptions {
	return func(opts *derivedOptions) {
		opts.labelKeys = keys
	}
}

// WithValueFrom is used to configure the derivation of a DerivedMetric. This option
// is mutually exclusive with the derived option `WithLabelKeys`. It acts as syntactic sugar
// that elides the need to create a DerivedMetric (with no labels) and then call `ValueFrom`.
func WithValueFrom(valueFn func() float64) DerivedOptions {
	return func(opts *derivedOptions) {
		opts.valueFn = valueFn
	}
}

//// Value creates a new LabelValue for the Label.
//func (l Label) Value(value string) LabelValue {
//	return tag.Upsert(tag.Key(l), value)
//}
//
//// MustCreateLabel will attempt to create a new Label. If
//// creation fails, then this method will panic.
//func MustCreateLabel(key string) Label {
//	k, err := NewKey(key)
//	if err != nil {
//		panic(fmt.Errorf("could not create label %q: %v", key, err))
//	}
//	return Label(k)
//}

// MustRegister is a helper function that will ensure that the provided Metrics are
// registered. If a metric fails to register, this method will panic.
func MustRegister(metrics ...Metric) {
	for _, m := range metrics {
		if err := m.Register(); err != nil {
			panic(err)
		}
	}
}

// RegisterIf is a helper function that will ensure that the provided
// Metric is registered if enabled function returns true.
// If a metric fails to register, this method will panic.
// It returns the registered metric or no-op metric based on enabled function.
// NOTE: It is important to use the returned Metric if RegisterIf is used.
func RegisterIf(metric Metric, enabled func() bool) Metric {
	if enabled() {
		if err := metric.Register(); err != nil {
			panic(err)
		}
		return metric
	}
	return &disabledMetric{name: metric.Name()}
}

// NewSum creates a new Metric with an aggregation type of Sum (the values will be cumulative).
// That means that data collected by the new Metric will be summed before export.
func NewSum(name, description string, opts ...Options) Metric {
	return newSum(name, description, opts...)
}

// NewGauge creates a new Metric with an aggregation type of LastValue. That means that data collected
// by the new Metric will export only the last recorded value.
func NewGauge(name, description string, opts ...Options) Metric {
	return newGauge(name, description, opts...)
}

// NewDerivedGauge creates a new Metric with an aggregation type of LastValue that generates the value
// dynamically according to the provided function. This can be used for values based on querying some
// state within a system (when event-driven recording is not appropriate).
//
// Only 2 usages (uptime and cache expiry in node agent) in istio
func NewDerivedGauge(name, description string, opts ...DerivedOptions) DerivedMetric {
	return newDerivedGauge(name, description, opts...)
}

// NewDistribution creates a new Metric with an aggregation type of Distribution. This means that the
// data collected by the Metric will be collected and exported as a histogram, with the specified bounds.
func NewDistribution(name, description string, bounds []float64, opts ...Options) Metric {
	return newDistribution(name, description, bounds, opts...)
}

var newSum func(name, description string, opts ...Options) Metric
var newGauge func(name, description string, opts ...Options) Metric
var newDistribution func(name, description string, bounds []float64, opts ...Options) Metric
var newDerivedGauge func(name, description string, opts ...DerivedOptions) DerivedMetric

func createOptions(opts ...Options) *options {
	o := &options{unit: None, labels: make([]string, 0)}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func createDerivedOptions(opts ...DerivedOptions) *derivedOptions {
	o := &derivedOptions{labelKeys: make([]string, 0)}
	for _, opt := range opts {
		opt(o)
	}
	// if a valueFn is supplied, then no label values can be supplied.
	// to prevent issues, drop the label keys
	if o.valueFn != nil {
		o.labelKeys = []string{}
	}
	return o
}
