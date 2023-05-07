//go:build !opencensus
// +build !opencensus

package monitoring

import "expvar"

func init() {
	newSum = newSumEV
	newGauge = newGaugeEV
	newDistribution = newDistributionEV
	newDerivedGauge = newDerivedGaugeEV
}

func newDerivedGaugeEV(name string, description string, opts ...DerivedOptions) DerivedMetric {
	return &disabledMetric{}
}

func newDistributionEV(name string, description string, bounds []float64, opts ...Options) Metric {
	return &disabledMetric{}
}

func newGaugeEV(name string, description string, opts ...Options) Metric {
	return &expvarInt{name: name}
}

func newSumEV(name string, description string, opts ...Options) Metric {
	return &disabledMetric{}
}

type expvarInt struct {
	name string
	expvar.Int
}

func (dm *expvarInt) ValueFrom(valueFn func() float64, labelValues ...string) {
}

// Decrement implements Metric
func (dm *expvarInt) Decrement() {}

// Increment implements Metric
func (dm *expvarInt) Increment() {}

// Name implements Metric
func (dm *expvarInt) Name() string {
	return dm.name
}

// Record implements Metric
func (dm *expvarInt) Record(value float64) {}

// RecordInt implements Metric
func (dm *expvarInt) RecordInt(value int64) {}

// Register implements Metric
func (dm *expvarInt) Register() error {
	return nil
}

// With implements Metric
func (dm *expvarInt) With(labelValues ...string) Metric {
	return dm
}

type expvarFloat struct {
	name string
}

func (dm *expvarFloat) ValueFrom(valueFn func() float64, labelValues ...string) {
}

// Decrement implements Metric
func (dm *expvarFloat) Decrement() {}

// Increment implements Metric
func (dm *expvarFloat) Increment() {}

// Name implements Metric
func (dm *expvarFloat) Name() string {
	return dm.name
}

// Record implements Metric
func (dm *expvarFloat) Record(value float64) {}

// RecordInt implements Metric
func (dm *expvarFloat) RecordInt(value int64) {}

// Register implements Metric
func (dm *expvarFloat) Register() error {
	return nil
}

// With implements Metric
func (dm *expvarFloat) With(labelValues ...string) Metric {
	return dm
}
