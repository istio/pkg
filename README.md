[![CircleCI](https://circleci.com/gh/istio/pkg.svg?style=svg)](https://circleci.com/gh/istio/pkg)
[![Mergify Status](https://gh.mergify.io/badges/istio/pkg.png?style=cut)](https://mergify.io)
[![codecov.io](https://codecov.io/gh/istio/pkg/branch/master/graph/badge.svg)](https://codecov.io/gh/istio/pkg)
[![Go Report Card](https://goreportcard.com/badge/github.com/istio/pkg)](https://goreportcard.com/report/github.com/istio/pkg)
[![GoDoc](https://godoc.org/istio.io/pkg?status.svg)](https://godoc.org/istio.io/pkg)
[![GolangCI](https://golangci.com/badges/github.com/istio/pkg.svg)](https://golangci.com/r/github.com/istio/pkg)

# Common Istio Packages

Common utility packages leveraged by other repos.

Packages in this repo are intended to expose fairly light-weight low-level abstractions.
In that vein, the repo in general maintains a fairly small number of external dependencies.

Of particular interest, packages in this repo shouldn't pull in Kubernetes dependencies.
