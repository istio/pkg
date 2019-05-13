lint:
	@scripts/linters.sh

format:
	@scripts/fmt.sh

gen:
	@go generate ./...
