all: truss

# Generate go files containing the all template files in []byte form
gobindata:
	go generate github.com/TuneLab/gob/...

# Install truss and all dependencies 
truss: gobindata
	go install github.com/TuneLab/gob/...

# Remove the truss binarys
clean:
	rm -f $(GOPATH)/bin/truss
	rm -f $(GOPATH)/bin/protoc-gen-truss-gokit
	rm -f $(GOPATH)/bin/protoc-gen-truss-doc

test:
	go test -tags=integration ./...

testclean:
	cd ./integration_tests && ./cleantest.sh



