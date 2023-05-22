[![codecov.io](https://codecov.io/gh/istio/pkg/branch/master/graph/badge.svg)](https://codecov.io/gh/istio/pkg)
[![Go Report Card](https://goreportcard.com/badge/github.com/istio/pkg)](https://goreportcard.com/report/github.com/istio/pkg)
[![GolangCI](https://golangci.com/badges/github.com/istio/pkg.svg)](https://golangci.com/r/github.com/istio/pkg)
[![GoDoc](https://godoc.org/istio.io/pkg?status.svg)](https://godoc.org/istio.io/pkg)

# Deprecation Notice

This repo has been merged into [istio.io/istio/pkg](https://github.com/istio/istio/blob/master/pkg/). Please go to that repo
to make any changes. The only exception is bug backports, which should be submitted here. The text
below is preserved for reference but is no longer maintained at this location.

# Common Istio Packages

Common utility packages leveraged by other repos.

Packages in this repo are intended to expose fairly light-weight low-level abstractions.
In that vein, the repo in general maintains a fairly small number of external dependencies.

Of particular interest, packages in this repo shouldn't pull in Kubernetes dependencies.
