all: generate build-proto build-manager build-manager-migration build-cosmos build-scheduler

.PHONY: generate
generate:
	go generate ./...

.PHONY: build
build:
	CGO_ENABLED="1" go build  -o worker ./cmd/terra-worker
