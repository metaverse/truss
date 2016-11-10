# Makefile for Truss.
#
# Build native Truss by default.
default: truss

dependencies: 
	go install github.com/golang/protobuf/protoc-gen-go

# Generate go files containing the all template files in []byte form
gobindata:
	go generate github.com/TuneLab/go-truss/gengokit/template
	go generate github.com/TuneLab/go-truss/truss/template

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
