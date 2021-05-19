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

package monitoring

import (
	"context"

	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricexport"
)

type MetricOperation func(operations)

type operations map[string]metricOperation

type metricOperation struct {
	newName       string
	labelNameMap  map[string]string
	labelValueMap map[string]LabelValueMapperFn
	constLabels   map[string]string
	dropLabels    []string
}

func RenameMetric(metric, newName string) MetricOperation {
	return func(ops operations) {
		op, found := ops[metric]
		if !found {
			mop := makeMetricOperation()
			mop.newName = newName
			ops[metric] = mop
			return
		}
		op.newName = newName
	}
}

func RenameLabels(metric string, labelMap map[string]string) MetricOperation {
	return func(ops operations) {
		op, found := ops[metric]
		if !found {
			mop := makeMetricOperation()
			mop.labelNameMap = labelMap
			ops[metric] = mop
			return
		}
		for k, v := range labelMap {
			op.labelNameMap[k] = v
		}
	}
}

type LabelValueMapperFn func(in string) string

func MapLabelValues(metric string, mapFns map[string]LabelValueMapperFn) MetricOperation {
	return func(ops operations) {
		op, found := ops[metric]
		if !found {
			mop := makeMetricOperation()
			mop.labelValueMap = mapFns
			ops[metric] = mop
			return
		}
		for k, v := range mapFns {
			op.labelValueMap[k] = v
		}
	}
}

func AddConstLabelValues(metric string, labels map[string]string) MetricOperation {
	return func(ops operations) {
		op, found := ops[metric]
		if !found {
			mop := makeMetricOperation()
			mop.constLabels = labels
			ops[metric] = mop
			return
		}
		for k, v := range labels {
			op.constLabels[k] = v
		}
	}
}

func DropLabels(metric string, toDrop []string) MetricOperation {
	return func(ops operations) {
		op, found := ops[metric]
		if !found {
			mop := makeMetricOperation()
			mop.dropLabels = toDrop
			ops[metric] = mop
			return
		}
		op.dropLabels = append(op.dropLabels, toDrop...)
	}
}

func makeMetricOperation() metricOperation {
	return metricOperation{
		labelNameMap:  make(map[string]string),
		labelValueMap: make(map[string]LabelValueMapperFn),
		constLabels:   make(map[string]string),
		dropLabels:    make([]string, 0),
	}
}

type transformingExporter struct {
	baseExporter     metricexport.Exporter
	metricOperations operations
	dropUnmodified   bool
}

func (te *transformingExporter) ExportMetrics(ctx context.Context, data []*metricdata.Metric) error {
	newData := make([]*metricdata.Metric, 0, len(data))
	for _, datum := range data {
		op, found := te.metricOperations[datum.Descriptor.Name]
		if !found {
			if !te.dropUnmodified {
				newData = append(newData, datum)
			}
			continue
		}
		newMetric := &metricdata.Metric{Resource: datum.Resource}
		newMetric.Descriptor = metricdata.Descriptor{
			Name:        datum.Descriptor.Name,
			Description: datum.Descriptor.Description,
			Unit:        datum.Descriptor.Unit,
			Type:        datum.Descriptor.Type,
			LabelKeys:   datum.Descriptor.LabelKeys,
		}
		newMetric.TimeSeries = datum.TimeSeries
		if op.newName != "" {
			newMetric.Descriptor.Name = op.newName
		}
		if len(op.labelNameMap) > 0 || len(op.constLabels) > 0 {
			newKeys := make([]metricdata.LabelKey, 0, len(datum.Descriptor.LabelKeys))
			for _, key := range datum.Descriptor.LabelKeys {
				newKey, found := op.labelNameMap[key.Key]
				if found {
					newKeys = append(newKeys, metricdata.LabelKey{Key: newKey, Description: key.Description})
				} else {
					newKeys = append(newKeys, key)
				}
			}
			for k, _ := range op.constLabels {
				newKeys = append(newKeys, metricdata.LabelKey{Key: k})
			}
			newMetric.Descriptor.LabelKeys = newKeys
		}
		if len(op.labelValueMap) > 0 || len(op.constLabels) > 0 {
			newTimeSeries := make([]*metricdata.TimeSeries, 0, len(datum.TimeSeries))
			for _, ts := range datum.TimeSeries {
				newTS := &metricdata.TimeSeries{StartTime: ts.StartTime, Points: ts.Points, LabelValues: make([]metricdata.LabelValue, 0, len(ts.LabelValues))}
				for i, lv := range ts.LabelValues {
					fn, found := op.labelValueMap[datum.Descriptor.LabelKeys[i].Key]
					if found {
						newTS.LabelValues = append(newTS.LabelValues, metricdata.LabelValue{Value: fn(lv.Value), Present: lv.Present})
					} else {
						newTS.LabelValues = append(newTS.LabelValues, lv)
					}
				}
				for _, v := range op.constLabels {
					newTS.LabelValues = append(newTS.LabelValues, metricdata.LabelValue{Value: v, Present: true})
				}
				newTimeSeries = append(newTimeSeries, newTS)
			}
			newMetric.TimeSeries = newTimeSeries
		}

		newData = append(newData, newMetric)
	}

	return te.baseExporter.ExportMetrics(ctx, newData)
}

func NewTransformingExporter(baseExporter metricexport.Exporter, dropUnmodified bool, ops ...MetricOperation) metricexport.Exporter {
	o := make(operations)
	for _, operation := range ops {
		operation(o)
	}
	return &transformingExporter{baseExporter: baseExporter, dropUnmodified: dropUnmodified, metricOperations: o}
}
