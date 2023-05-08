//go:build !skip_opencensus

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
	"context"
	"sync"

	"go.opencensus.io/metric"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricproducer"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"istio.io/pkg/log"
)

var (
	recordHookMutex sync.RWMutex
	recordHooks     map[string]RecordHook
	derivedRegistry = metric.NewRegistry()
)

// RecordHook has a callback function which a measure is recorded.
type RecordHook interface {
	OnRecordFloat64Measure(f *stats.Float64Measure, tags []tag.Mutator, value float64)
	OnRecordInt64Measure(i *stats.Int64Measure, tags []tag.Mutator, value int64)
}

func init() {
	recordHooks = make(map[string]RecordHook)
	// ensures exporters can see any derived metrics
	metricproducer.GlobalManager().AddProducer(derivedRegistry)
	newSum = newSumOC
	newGauge = newGaugeOC
	newDistribution = newDistributionOC
	newDerivedGauge = newDerivedGaugeOpenCensus
}

// RegisterRecordHook adds a RecordHook for a given measure.
func RegisterRecordHook(name string, h RecordHook) {
	recordHookMutex.Lock()
	defer recordHookMutex.Unlock()
	recordHooks[name] = h
}

// NewDistribution creates a new Metric with an aggregation type of Distribution. This means that the
// data collected by the Metric will be collected and exported as a histogram, with the specified bounds.
func newDistributionOC(name, description string, bounds []float64, opts ...Options) Metric {
	return newMetricOC(name, description, view.Distribution(bounds...), opts...)
}

func newSumOC(name, description string, opts ...Options) Metric {
	return newMetricOC(name, description, view.Sum(), opts...)
}

// NewGauge creates a new Metric with an aggregation type of LastValue. That means that data collected
// by the new Metric will export only the last recorded value.
func newGaugeOC(name, description string, opts ...Options) Metric {
	return newMetricOC(name, description, view.LastValue(), opts...)
}

func newMetricOC(name, description string, aggregation *view.Aggregation, opts ...Options) Metric {
	o := createOptions(opts...)
	return newFloat64Metric(name, description, aggregation, o)
}

func newDerivedGaugeOpenCensus(name, description string, opts ...DerivedOptions) DerivedMetric {
	options := createDerivedOptions(opts...)
	m, err := derivedRegistry.AddFloat64DerivedGauge(name,
		metric.WithDescription(description),
		metric.WithLabelKeys(options.labelKeys...),
		metric.WithUnit(metricdata.UnitDimensionless)) // TODO: allow unit in options
	if err != nil {
		log.Warnf("failed to add metric %q: %v", name, err)
	}
	derived := &derivedFloat64Metric{
		base: m,
		name: name,
	}
	//if options.valueFn != nil {
	//	derived.ValueFrom(options.valueFn)
	//}
	return derived
}

type derivedFloat64Metric struct {
	base *metric.Float64DerivedGauge

	name string
}

func (d *derivedFloat64Metric) Name() string {
	return d.name
}

// no-op
func (d *derivedFloat64Metric) Register() error {
	return nil
}

func (d *derivedFloat64Metric) ValueFrom(valueFn func() float64, labelValues ...Attr) {
	if len(labelValues) == 0 {
		if err := d.base.UpsertEntry(valueFn); err != nil {
			log.Errorf("failed to add value for derived metric %q: %v", d.name, err)
		}
		return
	}
	lv := make([]metricdata.LabelValue, 0, len(labelValues))
	for _, l := range labelValues {
		lv = append(lv, metricdata.NewLabelValue(l.Value))
	}
	if err := d.base.UpsertEntry(valueFn, lv...); err != nil {
		log.Errorf("failed to add value for derived metric %q: %v", d.name, err)
	}
}

type float64Metric struct {
	*stats.Float64Measure

	// tags stores all tags for the metrics
	tags []tag.Mutator
	// ctx is a precomputed context holding tags, as an optimization
	ctx  context.Context
	view *view.View

	incrementMeasure []stats.Measurement
	decrementMeasure []stats.Measurement
}

func newFloat64Metric(name, description string, aggregation *view.Aggregation, opts *options) *float64Metric {
	measure := stats.Float64(name, description, string(opts.unit))
	tagKeys := make([]tag.Key, 0, len(opts.labels))
	for _, l := range opts.labels {
		tagKeys = append(tagKeys, tag.MustNewKey(string(l)))
	}
	ctx, _ := tag.New(context.Background()) //nolint:errcheck
	return &float64Metric{
		Float64Measure:   measure,
		tags:             make([]tag.Mutator, 0),
		ctx:              ctx,
		view:             &view.View{Measure: measure, TagKeys: tagKeys, Aggregation: aggregation},
		incrementMeasure: []stats.Measurement{measure.M(1)},
		decrementMeasure: []stats.Measurement{measure.M(-1)},
	}
}

func (f *float64Metric) Increment() {
	f.recordMeasurements(f.incrementMeasure)
}

func (f *float64Metric) Decrement() {
	f.recordMeasurements(f.decrementMeasure)
}

func (f *float64Metric) Name() string {
	return f.Float64Measure.Name()
}

func (f *float64Metric) Record(value float64) {
	recordHookMutex.RLock()
	if rh, ok := recordHooks[f.Name()]; ok {
		rh.OnRecordFloat64Measure(f.Float64Measure, f.tags, value)
	}
	recordHookMutex.RUnlock()
	m := f.M(value)
	stats.Record(f.ctx, m) //nolint:errcheck
}

func (f *float64Metric) recordMeasurements(m []stats.Measurement) {
	recordHookMutex.RLock()
	if rh, ok := recordHooks[f.Name()]; ok {
		for _, mv := range m {
			rh.OnRecordFloat64Measure(f.Float64Measure, f.tags, mv.Value())
		}
	}
	recordHookMutex.RUnlock()
	stats.Record(f.ctx, m...)
}

func (f *float64Metric) RecordInt(value int64) {
	f.Record(float64(value))
}

// A LabelValue represents a Label with a specific value. It is used to record
// values for a Metric.
type LabelValue tag.Mutator

func toLabelValues(args ...Attr) []tag.Mutator {
	t := make([]tag.Mutator, len(args))
	for _, a := range args {
		t = append(t, tag.Insert(tag.MustNewKey(a.Key), a.Value))
	}
	return nil
}

func (f *float64Metric) With(labelValues ...Attr) Metric {
	t := make([]tag.Mutator, len(f.tags), len(f.tags)+len(labelValues))
	copy(t, f.tags)
	lv := toLabelValues(labelValues...)
	for _, tagValue := range lv {
		t = append(t, tagValue)
	}
	ctx, _ := tag.New(context.Background(), t...) //nolint:errcheck
	return &float64Metric{
		Float64Measure:   f.Float64Measure,
		tags:             t,
		ctx:              ctx,
		view:             f.view,
		incrementMeasure: f.incrementMeasure,
		decrementMeasure: f.decrementMeasure,
	}
}

func (f *float64Metric) Register() error {
	return view.Register(f.view)
}
