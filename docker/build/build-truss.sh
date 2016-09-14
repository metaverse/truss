#!/bin/sh
# Shell script to build Truss inside the docker container.
#
# This script expects environment variables that are set up inside of
# Dockerfile. It is only intended to be run from within the truss-build Docker
# container.
#
# Truss can be built without this shell script, but because creating the output
# from the truss build is a multi-step process, a shell script seemed a little
# cleaner than stringing together a bunch of commands in a Dockerfile CMD
# directive.

# Build the truss binary and our protoc plugin.
go build -o build/truss $TRUSS_REPO/truss
go build -o build/$PROTO_GEN_TRUSS $TRUSS_REPO/$PROTO_GEN_TRUSS

# Copy protoc and the protoc generator plugins for protoc to the output
# directory. (Protoc is downloaded when the build image is created.)
cp /bin/protoc $GOPATH/bin/protoc-gen-* $TRUSS_PATH/build

# In the output directory, create a tarball of the shared libraries that protoc
# demands at runtime.
cat /tmp/protoc-libs.txt | xargs tar czf $TRUSS_PATH/build/shared-libs.tgz -h -C /
