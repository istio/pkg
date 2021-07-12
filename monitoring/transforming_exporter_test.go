// Copyright 2021 Istio Authors
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

package monitoring_test

import (
	"errors"
	"fmt"
	"testing"

	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricexport"
	"go.opencensus.io/stats/view"
	"istio.io/pkg/monitoring"
)

func TestTransformExporter(t *testing.T) {
	exp := &testExporter{rows: make(map[string][]*view.Row), metrics: make(map[string][]*metricdata.Metric)}

	testSum.With(name.Value("foo"), kind.Value("bar")).Increment()
	goofySum.With(name.Value("baz")).Record(45)
	goofySum.With(name.Value("baz")).Decrement()

	testGauge.Record(42)
	testGauge.Record(77)

	testDerivedGauge := monitoring.NewDerivedGauge(
		"test_derived_gauge",
		"Testing derived gauae functionality",
		func() float64 {
			return 17.76
		},
	)

	reader := metricexport.NewReader()

	tranformingExporter := monitoring.NewTransformingExporter(exp,
		true, /* drop unmodified */
		monitoring.RenameMetric(testDerivedGauge.Name(), "funtime_happy_gauge"),
		monitoring.RenameLabels(testSum.Name(), map[string]string{"name": "aka"}),
		monitoring.MapLabelValues(testSum.Name(), map[string]monitoring.LabelValueMapperFn{
			"kind": func(in string) string { return "serious" },
		}),
		monitoring.AddConstLabelValues(testDerivedGauge.Name(), map[string]string{"version": "v1.2.3"}),
	)

	err := retry(
		func() error {
			reader.ReadAndExport(tranformingExporter)

			// check dropped
			if len(exp.metrics[testGauge.Name()]) != 0 {
				return fmt.Errorf("metric recorded for unmodifed gauge %q (that should have been dropped)", testGauge.Name())
			}

			// check gauge
			if len(exp.metrics["funtime_happy_gauge"]) < 1 {
				return fmt.Errorf("no metric recorded for name-mapped gauge %q", testDerivedGauge.Name())
			}
			for _, metric := range exp.metrics["funtime_happy_gauge"] {
				for _, ts := range metric.TimeSeries {
					for _, point := range ts.Points {
						if got, want := point.Value.(float64), 17.76; got != want {
							return fmt.Errorf("unexpected value for gauge; got %f, want %f", got, want)
						}
					}
				}
			}

			// check label renaming
			if len(exp.metrics[testSum.Name()]) < 1 {
				return fmt.Errorf("no metric recorded for dimensioned sum: %#v", exp.metrics)
			}
			for _, metric := range exp.metrics[testSum.Name()] {
				found := false
				for _, key := range metric.Descriptor.LabelKeys {
					if key.Key == "aka" {
						found = true
					}
				}
				if !found {
					return fmt.Errorf("renamed label not found")
				}
			}

			// check label value mapping
			for _, metric := range exp.metrics[testSum.Name()] {
				for _, ts := range metric.TimeSeries {
					found := false
					for _, lv := range ts.LabelValues {
						if lv.Value == "serious" {
							found = true
						}
					}
					if !found {
						return fmt.Errorf("mapped label value not found")
					}
				}
			}

			// check add const labels
			for _, metric := range exp.metrics["funtime_happy_gauge"] {
				found := false
				for _, lk := range metric.Descriptor.LabelKeys {
					if lk.Key == "version" {
						found = true
					}
				}
				if !found {
					return errors.New("could not find added const label key")
				}
				for _, ts := range metric.TimeSeries {
					found := false
					for _, lv := range ts.LabelValues {
						if lv.Value == "v1.2.3" {
							found = true
						}
					}
					if !found {
						return errors.New("could not find added const label value")
					}
				}
			}

			return nil
		})

	if err != nil {
		t.Fatalf("failure: %v", err)
	}
}
