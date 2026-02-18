GORELEASER_ARGS ?= --snapshot --clean

GUESSTIMATE_LATEST_VERSION ?= $(shell git describe --tags --abbrev=0)

build:
	CGO_ENABLED=0 go build -o bin/guesstimate ./cmd/guesstimate

release:
	goreleaser $(GORELEASER_ARGS)