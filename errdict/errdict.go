package errdict

import "istio.io/pkg/monitoring"

type IstioErrorStruct struct {
	MoreInfo     string
	Impact       string
	Action       string
	LikelyCauses string

	// Metrics is a list of associated metrics that will be passed to a handler from the call site.
	Metrics []monitoring.Metric
}
