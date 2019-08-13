export ARTIFACTS ?= "/tmp"

gen:
	@go generate ./...

lint:
	# These PATH hacks are temporary until prow properly sets its paths
	@PATH=${PATH}:${GOPATH}/bin scripts/check_license.sh
	@PATH=${PATH}:${GOPATH}/bin scripts/run_golangci.sh

fmt:
	@scripts/run_gofmt.sh

test:
	@go test -race ./...

test_with_coverage:
	@go test -race -coverprofile=${ARTIFACTS}/coverage.cov -covermode=atomic ./...
	@curl -s https://codecov.io/bash | bash -s -- -c -F aFlag -f ${ARTIFACTS}/coverage.cov

include Makefile.common.mk
