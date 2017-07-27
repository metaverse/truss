# Makefile for Truss.
#

# Pre-install required binaries
$(shell go get github.com/golang/protobuf/protoc-gen-go)
$(shell go get github.com/jteeuwen/go-bindata/...)
$(shell go get github.com/pauln/go-datefmt)

# Get version Date
SEMVER := $(shell cat VERSION)
TAG_COMMIT := $(shell git rev-list -n 1 ${SEMVER})
GIT_COMMIT_EPOC := $(shell git show -s --format=%ct ${TAG_COMMIT})
VERSION_DATE := $(shell go-datefmt -ts ${GIT_COMMIT_EPOC} -fmt UnixDate -utc)

# Build native Truss by default.
default: truss

dependencies:
	glide install

update-dependencies:
	glide update

# Generate go files containing the all template files in []byte form
gobindata:
	go generate github.com/TuneLab/truss/gengokit/template

# Install truss and protoc-gen-truss-protocast
truss: gobindata dependencies
	go install github.com/TuneLab/truss/cmd/protoc-gen-truss-protocast
	go install -ldflags '-X "main.Version=$(SEMVER)" -X "main.VersionDate=$(VERSION_DATE)"' github.com/TuneLab/truss/cmd/truss

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
