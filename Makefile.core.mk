# Copyright 2019 Istio Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

export ARTIFACTS ?= "/tmp"

gen:
	@go generate ./...

lint: lint-all

fmt: format-go

build:
	@go build ./...

test:
	@go test -race -coverprofile=${ARTIFACTS}/coverage.cov -covermode=atomic ./...
	@curl -s https://codecov.io/bash | bash -s -- -c -F aFlag -f ${ARTIFACTS}/coverage.cov

test_with_coverage: test

include common/Makefile.common.mk
