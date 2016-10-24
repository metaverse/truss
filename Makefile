# Makefile for Truss.
#
# Build native Truss by default.
default: truss

# Generate go files containing the all template files in []byte form
gobindata:
	go generate github.com/TuneLab/go-truss/gengokit/template
	go generate github.com/TuneLab/go-truss/truss/template

# Install truss and protoc-gen-truss-protocast
truss: gobindata
	go install github.com/TuneLab/go-truss/protoc-gen-truss-protocast
	go install github.com/TuneLab/go-truss/truss

# Run the go tests and the truss integration tests
test: test-go test-integration

test-go:
	go test -v ./...

test-integration:
	$(MAKE) -C truss test-integration

# Run the go non-vendored unit tests
test-nv:
	go test -v ./deftree/... ./gendoc/... ./gengokit/... \
		./protoc-gen-truss-protocast/... ./truss/...

# Removes generated code from tests
testclean:
	$(MAKE) -C truss testclean

# Build Truss and package it in a Docker container, according to the rules
# in docker/Makefile.
docker:
	$(MAKE) -C docker

.PHONY: docker
