# Makefile for Truss.
#
SHA := $(shell git rev-parse --short=10 HEAD)

MAKEFILE_PATH := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
VERSION_DATE := $(shell $(MAKEFILE_PATH)/commit_date.sh)

# Build native Truss by default.
default: truss

dependencies:
	go get github.com/golang/protobuf/protoc-gen-go
	go get github.com/golang/protobuf/proto
	go get github.com/jteeuwen/go-bindata/...

update-dependencies:
	go get -u github.com/golang/protobuf/protoc-gen-go
	go get -u github.com/golang/protobuf/proto
	go get -u github.com/jteeuwen/go-bindata/...

# Generate go files containing the all template files in []byte form
gobindata:
	go generate github.com/TuneLab/go-truss/gengokit/template

# Install truss and protoc-gen-truss-protocast
truss: gobindata
	go install github.com/TuneLab/go-truss/cmd/protoc-gen-truss-protocast
	go install -ldflags '-X "main.Version=$(SHA)" -X "main.BuildDate=$(VERSION_DATE)"' github.com/TuneLab/go-truss/cmd/truss

# Run the go tests and the truss integration tests
test: test-go test-integration

test-go:
	go test -v ./...

test-integration:
	$(MAKE) -C cmd/_integration-tests

# Removes generated code from tests
testclean:
	$(MAKE) -C cmd/_integration-tests clean


.PHONY: testclean test-integration test-go test truss gobindata dependencies
