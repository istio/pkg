lint:
	@scripts/linters.sh

fixlint:
	@scripts/linters.sh --fix

format:
	@scripts/fmt.sh

gen:
	@go generate ./...
