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

package metrics_test

import (
	"reflect"
	"testing"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"istio.io/pkg/collateral/metrics"
)

func TestExportedMetrics(t *testing.T) {
	registerViews()
	r := metrics.NewOpenCensusRegistry()
	if got := r.ExportedMetrics(); !reflect.DeepEqual(got, want) {
		t.Errorf("ExportedMetrics() = %v, want %v", got, want)
	}
}

// stolen shamelessly from mixer pkgs in istio/istio for testing purposes
var (
	// AttributesTotal is a measure of the number of known attributes.
	AttributesTotal = stats.Int64(
		"mixer/config/attributes_total",
		"The number of known attributes in the current config.",
		stats.UnitDimensionless)

	// HandlersTotal is a measure of the number of known handlers.
	HandlersTotal = stats.Int64(
		"mixer/config/handler_configs_total",
		"The number of known handlers in the current config.",
		stats.UnitDimensionless)

	// InstancesTotal is a measure of the number of known instances.
	InstancesTotal = stats.Int64(
		"mixer/config/instance_configs_total",
		"The number of known instances in the current config.",
		stats.UnitDimensionless)

	// InstanceErrs is a measure of the number of errors for processing instance config.
	InstanceErrs = stats.Int64(
		"mixer/config/instance_config_errors_total",
		"The number of errors encountered during processing of the instance configuration.",
		stats.UnitDimensionless)

	// RulesTotal is a measure of the number of known rules.
	RulesTotal = stats.Int64(
		"mixer/config/rule_configs_total",
		"The number of known rules in the current config.",
		stats.UnitDimensionless)

	// RuleErrs is a measure of the number of errors for processing rules config.
	RuleErrs = stats.Int64(
		"mixer/config/rule_config_errors_total",
		"The number of errors encountered during processing of the rule configuration.",
		stats.UnitDimensionless)

	// AdapterInfosTotal is a measure of the number of known adapters.
	AdapterInfosTotal = stats.Int64(
		"mixer/config/adapter_info_configs_total",
		"The number of known adapters in the current config.",
		stats.UnitDimensionless)

	want = []metrics.Exported{
		{"mixer_config_adapter_info_configs_total", "LastValue", "The number of known adapters in the current config."},
		{"mixer_config_attributes_total", "LastValue", "The number of known attributes in the current config."},
		{"mixer_config_handler_configs_total", "LastValue", "The number of known handlers in the current config."},
		{"mixer_config_instance_config_errors_total", "LastValue", "The number of errors encountered during processing of the instance configuration."},
		{"mixer_config_instance_configs_total", "LastValue", "The number of known instances in the current config."},
		{"mixer_config_rule_config_errors_total", "LastValue", "The number of errors encountered during processing of the rule configuration."},
		{"mixer_config_rule_configs_total", "LastValue", "The number of known rules in the current config."},
	}
)

func newView(measure stats.Measure, keys []tag.Key, aggregation *view.Aggregation) *view.View {
	return &view.View{
		Name:        measure.Name(),
		Description: measure.Description(),
		Measure:     measure,
		TagKeys:     keys,
		Aggregation: aggregation,
	}
}

func registerViews() {
	views := []*view.View{
		// config views
		newView(AttributesTotal, []tag.Key{}, view.LastValue()),
		newView(HandlersTotal, []tag.Key{}, view.LastValue()),
		newView(InstancesTotal, []tag.Key{}, view.LastValue()),
		newView(InstanceErrs, []tag.Key{}, view.LastValue()),
		newView(RulesTotal, []tag.Key{}, view.LastValue()),
		newView(RuleErrs, []tag.Key{}, view.LastValue()),
		newView(AdapterInfosTotal, []tag.Key{}, view.LastValue()),
	}

	view.Register(views...)
}
