Istio Performance Analysis

The goal of this experiment is to measure the amount of latency that is added to a simple service by the addition of various Istio Components.  To isolate the contribution of each component, many combinations of avialable components were tested and recorded.

Components
 * Sidecar
 * Istio Ingress
 * Load Balancer Ingress
 
Other Parameters
 * Scale: 1 or 5 instances of services and ingress
 * Service: Single HTTP server or frontend/backend combo
 
Methodology
 
Each combination of Parameters was tested using a fortio client running inside the cluster, with 64 concurrent connections for 30 seconds.  First the max QPS was measured (using -qps 0), then a second test was run at 75% load as per fortio recommendations, and the P50 and P99 latencies were observed
 