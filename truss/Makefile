all: truss

# Generate go files containing the all template files in []byte form
gobindata:
	go generate github.com/TuneLab/gob/...

# Install truss and all dependencies 
truss: gobindata
	go install github.com/TuneLab/gob/...

# Remove the truss binarys
clean:
	rm -r $(GOPATH)/bin/truss
	rm -r $(GOPATH)/bin/protoc-gen-truss-gokit
	rm -r $(GOPATH)/bin/protoc-gen-truss-docs
	rm -r $(GOPATH)/bin/truss-integration-runner

# Run truss on all proto files in all dirs in ./tests/
test:
	cd ./tests && truss-integration-runner

# Remove the dir named `service/` in all dirs in ./tests/
testclean:
	./tests/cleantest.sh ./tests/


