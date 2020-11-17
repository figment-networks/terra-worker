LDFLAGS      := -w -s
MODULE       := github.com/figment-networks/terra-worker
VERSION_FILE ?= ./VERSION

# Git Status
GIT_SHA ?= $(shell git rev-parse --short HEAD)

ifneq (,$(wildcard $(VERSION_FILE)))
VERSION ?= $(shell head -n 1 $(VERSION_FILE))
else
VERSION ?= n/a
endif

all: generate pack-release

.PHONY: generate
generate:
	go generate ./...

.PHONY: plugin
plugin:
	CGO_ENABLED="1" go build -trimpath -o converter-plugin.so -buildmode=plugin ./cmd/converter-plugin

.PHONY: build
build: LDFLAGS += -X $(MODULE)/cmd/terra-worker/config.Timestamp=$(shell date +%s)
build: LDFLAGS += -X $(MODULE)/cmd/terra-worker/config.Version=$(VERSION)
build: LDFLAGS += -X $(MODULE)/cmd/terra-worker/config.GitSHA=$(GIT_SHA)
build:
	CGO_ENABLED="1" go build -o worker -ldflags '$(LDFLAGS)'  ./cmd/terra-worker

.PHONY: pack-release
pack-release:
	@mkdir -p ./release
	@make build
	@mv ./worker ./release/worker
	@zip -r terra-worker ./release
	@rm -rf ./release

.PHONY: pack-release-with-libs
pack-release-with-libs:
	@mkdir -p ./release
	@go mod vendor
	@make build
	@mv ./worker ./release/worker
	@cp ./vendor/github.com/CosmWasm/go-cosmwasm/api/libgo_cosmwasm.so ./release/libgo_cosmwasm.so
	@zip -r terra-worker ./release
	@rm -rf ./vendor
	@rm -rf ./release

