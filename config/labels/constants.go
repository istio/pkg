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

package labels

const (
	// IstioLocality indicates the region/zone/subzone of an instance. It is used to override the native
	// registry's value.
	//
	// Note: because k8s labels does not support `/`, so we use `.` instead in k8s.
	IstioLocality = "istio-locality"

	// Istio indicates that a workload is part of a named Istio system component.
	Istio = "istio"

	// IngressValue is value for the Istio label that identifies an ingress workload.
	IngressValue = "ingress"
)
