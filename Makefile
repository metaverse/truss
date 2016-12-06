# Makefile for Truss.
#
# Build native Truss by default.
default: truss

dependencies:
	go get github.com/go-kit/kit
	go get google.golang.org/genproto
	go get github.com/golang/protobuf/{proto,protoc-gen-go}

update-dependencies:
	go get -u github.com/go-kit/kit
	go get -u google.golang.org/genproto
	go get -u github.com/golang/protobuf/{proto,protoc-gen-go}

# Generate go files containing the all template files in []byte form
gobindata:
	go generate github.com/TuneLab/go-truss/gengokit/template

# Install truss and protoc-gen-truss-protocast
truss: gobindata
	go install github.com/TuneLab/go-truss/cmd/protoc-gen-truss-protocast
	go install github.com/TuneLab/go-truss/cmd/truss

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
