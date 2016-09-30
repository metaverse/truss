all: truss

# Generate go files containing the all template files in []byte form
gobindata:
	go generate github.com/TuneLab/go-truss/...

# Install truss and all dependencies 
truss: gobindata
	go install github.com/TuneLab/go-truss/...

# Remove the truss binaries
clean:
	rm -f $(GOPATH)/bin/truss
	rm -f $(GOPATH)/bin/protoc-gen-truss-protocast

test: testclean
	$(MAKE) -C _integration-tests

testclean:
	$(MAKE) -C _integration-tests clean
