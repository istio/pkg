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

package monitoring_test

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"istio.io/pkg/monitoring"
)

var (
	name = monitoring.MustCreateLabel("name")
	kind = monitoring.MustCreateLabel("kind")

	testSum = monitoring.NewSum(
		"events_total",
		"Number of events observed, by name and kind",
		monitoring.WithLabels(name, kind),
	)

	goofySum = testSum.With(kind.Value("goofy"))

	testDistribution = monitoring.NewDistribution(
		"test_buckets",
		"Testing distribution functionality",
		[]float64{0, 2.5, 7, 8, 10, 154.3, 99},
		monitoring.WithLabels(name),
		monitoring.WithUnit(monitoring.Seconds),
	)

	testGauge = monitoring.NewGauge(
		"test_gauge",
		"Testing gauge functionality",
	)
)

func init() {
	monitoring.MustRegister(testSum, testDistribution, testGauge)
}

func TestSum(t *testing.T) {
	exp := &testExporter{rows: make(map[string][]*view.Row)}
	view.RegisterExporter(exp)
	view.SetReportingPeriod(1 * time.Millisecond)

	testSum.With(name.Value("foo"), kind.Value("bar")).Increment()
	goofySum.With(name.Value("baz")).Record(45)
	goofySum.With(name.Value("baz")).Decrement()

	time.Sleep(2 * time.Millisecond)

	exp.Lock()
	if len(exp.rows[testSum.Name()]) < 2 {
		// we should have two values goofySum (which is a dimensioned testSum) and
		// testSum.
		t.Error("no values recorded for sum, want 2.")
	}
	for _, r := range exp.rows[testSum.Name()] {
		if findTagWithValue("kind", "goofy", r.Tags) {
			if sd, ok := r.Data.(*view.SumData); ok {
				if got, want := sd.Value, 44.0; got != want {
					t.Errorf("bad value for %q: %f, want %f", goofySum.Name(), got, want)
				}
			}
		} else if findTagWithValue("kind", "bar", r.Tags) {
			if sd, ok := r.Data.(*view.SumData); ok {
				if got, want := sd.Value, 1.0; got != want {
					t.Errorf("bad value for %q: %f, want %f", testSum.Name(), got, want)
				}
			}
		} else {
			t.Errorf("unknown row in results: %v", r)
		}
	}
	exp.Unlock()
}

func TestGauge(t *testing.T) {
	exp := &testExporter{rows: make(map[string][]*view.Row)}
	view.RegisterExporter(exp)
	view.SetReportingPeriod(1 * time.Millisecond)

	testGauge.Record(42)
	testGauge.Record(77)

	time.Sleep(2 * time.Millisecond)

	exp.Lock()
	// only last value should be kept
	if len(exp.rows[testGauge.Name()]) < 1 {
		t.Error("no values recorded for gauge, want 1.")
	}
	for _, r := range exp.rows[testGauge.Name()] {
		if lvd, ok := r.Data.(*view.LastValueData); ok {
			if got, want := lvd.Value, 77.0; got != want {
				t.Errorf("bad value for %q: %f, want %f", testGauge.Name(), got, want)
			}
		}
	}
	exp.Unlock()
}

func TestDistribution(t *testing.T) {
	exp := &testExporter{rows: make(map[string][]*view.Row)}
	view.RegisterExporter(exp)
	view.SetReportingPeriod(1 * time.Millisecond)

	funDistribution := testDistribution.With(name.Value("fun"))
	funDistribution.Record(7.7773)
	testDistribution.With(name.Value("foo")).Record(7.4)
	testDistribution.With(name.Value("foo")).Record(6.8)
	testDistribution.With(name.Value("foo")).Record(10.2)

	time.Sleep(2 * time.Millisecond)

	exp.Lock()
	if len(exp.rows[testDistribution.Name()]) < 2 {
		t.Error("no values recorded for distribution, want 2.")
	}

	for _, r := range exp.rows[testDistribution.Name()] {
		if findTagWithValue("name", "fun", r.Tags) {
			if dd, ok := r.Data.(*view.DistributionData); ok {
				if got, want := dd.Count, int64(1); got != want {
					t.Errorf("bad count for %q: %d, want %d", testDistribution.Name(), got, want)
				}
			}
		} else if findTagWithValue("name", "foo", r.Tags) {
			if dd, ok := r.Data.(*view.DistributionData); ok {
				if got, want := dd.Count, int64(3); got != want {
					t.Errorf("bad count for %q: %d, want %d", testDistribution.Name(), got, want)
				}
			}
		} else {
			t.Error("expected distributions not found.")
		}
	}
	exp.Unlock()
}

func TestMustCreateLabel(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if !strings.Contains(r.(error).Error(), "label") {
				t.Errorf("no panic for invalid label, recovered: %q", r.(error).Error())
			}
		} else {
			t.Error("no panic for failed label creation.")
		}
	}()

	// labels must be ascii
	monitoring.MustCreateLabel("£®")
}

func TestMustRegister(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("no panic for failed registration.")
		}
	}()

	monitoring.MustRegister(&registerFail{})
}

func TestViewExport(t *testing.T) {
	exp := &testExporter{rows: make(map[string][]*view.Row)}
	view.RegisterExporter(exp)
	view.SetReportingPeriod(1 * time.Millisecond)

	testSum.With(name.Value("foo"), kind.Value("bar")).Increment()
	goofySum.With(name.Value("baz")).Record(45)

	time.Sleep(2 * time.Millisecond)

	exp.Lock()
	if exp.invalidTags {
		t.Error("view registration includes invalid tag keys")
	}
	exp.Unlock()
}

type registerFail struct {
	monitoring.Metric
}

func (r registerFail) Register() error {
	return errors.New("fail")
}

type testExporter struct {
	sync.Mutex

	rows        map[string][]*view.Row
	invalidTags bool
}

func (t *testExporter) ExportView(d *view.Data) {
	t.Lock()
	for _, tk := range d.View.TagKeys {
		if len(tk.Name()) < 1 {
			t.invalidTags = true
		}
	}
	t.rows[d.View.Name] = append(t.rows[d.View.Name], d.Rows...)
	t.Unlock()
}

func findTagWithValue(key, value string, tags []tag.Tag) bool {
	for _, t := range tags {
		if t.Key.Name() == key && t.Value == value {
			return true
		}
	}
	return false
}
