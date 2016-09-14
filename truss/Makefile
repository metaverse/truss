all: truss

# Generate go files containing the all template files in []byte form
gobindata:
	go generate github.com/TuneLab/go-truss/...

# Install truss and all dependencies 
truss: gobindata
	go install github.com/TuneLab/go-truss/...

# Remove the truss binaries
clean:
	@rm -f $(GOPATH)/bin/truss \
		$(GOPATH)/bin/protoc-gen-truss-gokit
		$(GOPATH)/bin/protoc-gen-truss-doc \
		$(GOPATH)/bin/truss-integration-runner

# Run truss on all proto files in all dirs in ./tests/
test:
	go test -tags=integration -v ./integration_tests

testclean:
	go test -tags=integration ./integration_tests -clean


