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

// OpenCensus independent metrics

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
		// TODO: internal use only, make private
		Name() string

		// Record makes an observation of the provided value for the given measure.
		// Majority of Istio is setting this to an int
		Record(value float64)

		// RecordInt makes an observation of the provided value for the measure.
		// Not actually used in Istio.
		//RecordInt(value int64)

		// With creates a new Metric, with the LabelValues provided. This allows creating
		// a set of pre-dimensioned data for recording purposes. This is primarily used
		// for documentation and convenience. Metrics created with this method do not need
		// to be registered (they share the registration of their parent Metric).
		With(labelValues ...Attr) Metric

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
		ValueFrom(valueFn func() float64, labelValues ...Attr)
	}

	disabledMetric struct {
		name string
	}

	// Options encode changes to the options passed to a Metric at creation time.
	Options func(*options)

	// DerivedOptions encode changes to the options passed to a DerivedMetric at creation time.
	DerivedOptions func(*derivedOptions)

	options struct {
		unit   Unit
		labels []Label // Label
	}

	derivedOptions struct {
		labelKeys []string
	}
)

func (dm *disabledMetric) ValueFrom(func() float64, ...Attr) {
}

func (dm *disabledMetric) Increment() {}

func (dm *disabledMetric) Name() string {
	return dm.name
}

func (dm *disabledMetric) Record(value float64) {}

func (dm *disabledMetric) RecordInt(value int64) {}

func (dm *disabledMetric) Register() error {
	return nil
}

func (dm *disabledMetric) With(labelValues ...Attr) Metric {
	return dm
}

var _ Metric = &disabledMetric{}

// WithLabels provides configuration options for a new Metric, providing the expected
// dimensions for data collection for that Metric.
func WithLabels(labels ...Label) Options {
	return func(opts *options) {
		opts.labels = labels
	}
}

// WithUnit provides configuration options for a new Metric, providing unit of measure
// information for a new Metric.
// Used only 2x - once the type is part of the name ( as recommended), once is not.
// TODO: get rid of it or use consistently in ALL metrics.
func WithUnit(unit Unit) Options {
	return func(opts *options) {
		opts.unit = unit
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

// Label is used to ease migration from opencensus. Will be eventually replaced
// with the otel attributes, but in a second stage.
type Label string

// Attr is used to ease migration and minimize changes. Will be replaced by otel attributes.
type Attr struct {
	Key   string
	Value string
}

// MustCreateLabel is a temporary method to ease migration.
func MustCreateLabel(key string) Label {
	return Label(key)
}

func (l Label) Value(v string) Attr {
	return Attr{
		Key:   string(l),
		Value: v,
	}
}

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
// Per prom conventions, must have a name ending in _total.
//
// Istio doesn't do this for:
//
// num_outgoing_retries
// pilot_total_rejected_configs
// provider_lookup_cluster_failures
// xds_cache_reads
// xds_cache_evictions
// pilot_k8s_cfg_events
// pilot_k8s_reg_events
// pilot_k8s_endpoints_with_no_pods
// pilot_total_xds_rejects
// pilot_xds_expired_nonce
// pilot_xds_write_timeout
// pilot_xds_pushes
// pilot_push_triggers
// pilot_xds_push_context_errors
// pilot_total_xds_internal_errors
// pilot_inbound_updates
// wasm_cache_lookup_count
// wasm_remote_fetch_count
// wasm_config_conversion_count
// citadel_server_csr_count
// citadel_server_authentication_failure_count
// citadel_server_csr_parsing_err_count
// citadel_server_id_extraction_err_count
// citadel_server_csr_sign_err_count
// citadel_server_success_cert_issuance_count
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

// Internal methods used to hook one of the conditionally compiled implementations.
var newSum func(name, description string, opts ...Options) Metric
var newGauge func(name, description string, opts ...Options) Metric
var newDistribution func(name, description string, bounds []float64, opts ...Options) Metric
var newDerivedGauge func(name, description string, opts ...DerivedOptions) DerivedMetric

func createOptions(opts ...Options) *options {
	o := &options{unit: None, labels: make([]Label, 0)}
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
	return o
}
