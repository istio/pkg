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

import (
	"strings"
)

// Gets the locality from the labels, or falls back to to a default locality if not found
// Because Kubernetes labels don't support `/`, we replace "." with "/" as a workaround
func GetLocalityOrDefault(defaultLocality string, l Instance) string {
	if l != nil && l[IstioLocality] != "" {
		// if there are /'s present we don't need to replace
		if strings.Contains(l[IstioLocality], "/") {
			return l[IstioLocality]
		}
		// replace "." with "/"
		return strings.Replace(l[IstioLocality], ".", "/", -1)
	}
	return defaultLocality
}
