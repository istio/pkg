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

package metrics

import (
	"sort"
	"strings"
	"time"

	"go.opencensus.io/stats/view"
)

const (
	tableHeader = `<h2 id=\"metrics\">Exported Metrics</h2>
<table class=\"metrics\">
<thead>
<tr><th>Name</th><th>Type</th><th>Description</th></tr>
</thead>
<tbody>
`
	tableFooter = `</tbody>
</table>
`
)

// openCensusHTMLGenerator implements the metrics.HTMLGenerator interface.
// It should only be used to generate collateral for processes offline.
// It is not suitable for production usage.
type openCensusHTMLGenerator struct {
	nameDescMap map[string]string
	nameTypeMap map[string]string
}

// NewOpenCensusHTMLGenerator builds a new HTMLGenerator based on OpenCensus
// metrics export. As part of the setup, it configures the OpenCensus mechanisms
// for rapid reporting (1ms) and sleeps for double that period (2ms) to ensure
// an export happens before generation.
func NewOpenCensusHTMLGenerator() HTMLGenerator {
	// note: only use this for collateral generation
	// this reporting period is NOT suitable for all exporters
	view.SetReportingPeriod(1 * time.Millisecond)

	e := &openCensusHTMLGenerator{
		nameDescMap: make(map[string]string),
		nameTypeMap: make(map[string]string),
	}
	view.RegisterExporter(e)

	time.Sleep(2 * time.Millisecond) // allow export to happen
	return e
}

// ExportView implements view.Exporter
func (e *openCensusHTMLGenerator) ExportView(d *view.Data) {
	e.nameDescMap[d.View.Name] = d.View.Description
	e.nameTypeMap[d.View.Name] = d.View.Aggregation.Type.String()
}

// GenerateHTML implements metrics.HTMLGenerator.
// It emits a HTML string with all of the OpenCensus exported metrics
// listed in a table by name, type, and description.
func (e *openCensusHTMLGenerator) GenerateHTML() string {
	var sb strings.Builder

	sb.WriteString(tableHeader)

	names := []string{}
	for key := range e.nameDescMap {
		names = append(names, key)
	}

	sort.Strings(names)

	for _, n := range names {

		var d, t string
		if desc, ok := e.nameDescMap[n]; !ok {
			d = "N/A"
		} else {
			d = desc
		}

		if kind, ok := e.nameTypeMap[n]; !ok {
			t = view.Sum().Type.String()
		} else {
			t = kind
		}

		sb.WriteString("<tr><td>" + promName(n) + "</td>")
		sb.WriteString("<td>" + t + "</td>")
		sb.WriteString("<td>" + d + "</td></tr>\n")
	}

	sb.WriteString(tableFooter)

	return sb.String()
}

var charReplacer = strings.NewReplacer("/", "_", ".", "_", " ", "_", "-", "")

func promName(metricName string) string {
	s := strings.TrimPrefix(metricName, "/")
	return charReplacer.Replace(s)
}
