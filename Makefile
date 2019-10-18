# Makefile for Truss.
#
SHA := $(shell git rev-parse --short=10 HEAD)

MAKEFILE_PATH := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
VERSION_DATE := $(shell $(MAKEFILE_PATH)/commit_date.sh)

# Build native Truss by default.
default: truss

dependencies:
	go get -u github.com/gogo/protobuf/protoc-gen-gogo@21df5aa0e680850681b8643f0024f92d3b09930c
	go get -u github.com/gogo/protobuf/protoc-gen-gogofaster@21df5aa0e680850681b8643f0024f92d3b09930c
	go get -u github.com/gogo/protobuf/proto@21df5aa0e680850681b8643f0024f92d3b09930c
	go get -u github.com/kevinburke/go-bindata/go-bindata

# Generate go files containing the all template files in []byte form
gobindata:
	go generate github.com/metaverse/truss/gengokit/template

# Install truss
truss: gobindata
	go install -ldflags '-X "main.Version=$(SHA)" -X "main.VersionDate=$(VERSION_DATE)"' github.com/metaverse/truss/cmd/truss

# Run the go tests and the truss integration tests
test: test-go test-integration

test-go:
	GO111MODULE=on go test -v ./...

test-integration:
	GO111MODULE=on $(MAKE) -C cmd/_integration-tests

# Removes generated code from tests
testclean:
	$(MAKE) -C cmd/_integration-tests clean

.PHONY: testclean test-integration test-go test truss gobindata dependencies
