package errdict

import "istio.io/pkg/monitoring"

// IstioErrorStruct represents structured error information, for optional use in scope.X or log.X calls.
// See https://docs.google.com/document/d/1vdYswLQuYnrLA2fDjk6OoZx2flBABa18UjCGTn8gsg8/ for additional information.
type IstioErrorStruct struct {
	// MoreInfo is additional information about the error e.g. a link to context describing the context for the error.
	MoreInfo string
	// Impact is the likely impact of the error on system function e.g. "Proxies are unable to communicate with Istiod."
	Impact string
	// Action is the next step the user should take e.g. "Open an issue or bug report."
	Action string
	// LikelyCause is the likely cause for the error e.g. "Software bug."
	LikelyCause string

	// Metrics is a list of associated metrics that will be passed to a handler from the call site.
	Metrics []monitoring.Metric
}
