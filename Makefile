# Makefile for Truss.
#
# Build native Truss by default.
default:
	$(MAKE) -C truss

# Build Truss and package it in a Docker container, according to the rules
# in docker/Makefile.
docker:
	$(MAKE) -C docker

.PHONY: docker
