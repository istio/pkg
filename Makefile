gen:
	@go generate ./...

lint:
	@scripts/run_golangci.sh
	@scripts/check_license.sh
	@scripts/check_dockerfiles.sh

fmt:
	@scripts/run_gofmt.sh

include Makefile.common.mk
