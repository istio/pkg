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
	"testing"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"istio.io/pkg/collateral/metrics"
)

func TestGenerateHTML(t *testing.T) {
	registerViews()
	e := metrics.NewOpenCensusHTMLGenerator()

	if got := e.GenerateHTML(); got != expected {
		t.Errorf("GenerateHTML() = %v, want %v", got, expected)
	}
}

// stolen shamelessly from mixer pkgs in istio/istio for testing purposes
var (
	expected = `<h2 id=\"metrics\">Exported Metrics</h2>
<table class=\"metrics\">
<thead>
<tr><th>Name</th><th>Type</th><th>Description</th></tr>
</thead>
<tbody>
<tr><td>mixer_config_adapter_info_configs_total</td><td>LastValue</td><td>The number of known adapters in the current config.</td></tr>
<tr><td>mixer_config_attributes_total</td><td>LastValue</td><td>The number of known attributes in the current config.</td></tr>
<tr><td>mixer_config_handler_configs_total</td><td>LastValue</td><td>The number of known handlers in the current config.</td></tr>
<tr><td>mixer_config_instance_config_errors_total</td><td>LastValue</td><td>The number of errors encountered during processing of the instance configuration.</td></tr>
<tr><td>mixer_config_instance_configs_total</td><td>LastValue</td><td>The number of known instances in the current config.</td></tr>
<tr><td>mixer_config_rule_config_errors_total</td><td>LastValue</td><td>The number of errors encountered during processing of the rule configuration.</td></tr>
<tr><td>mixer_config_rule_configs_total</td><td>LastValue</td><td>The number of known rules in the current config.</td></tr>
</tbody>
</table>
`

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
