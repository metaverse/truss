# Makefile for Truss.
#
# Build native Truss by default.
default:
	$(MAKE) -C truss

# Run the go tests and the truss integration tests
test:
	go test -v ./...
	$(MAKE) -C truss test

testclean: 
	$(MAKE) -C truss testclean
