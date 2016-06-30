#!/usr/bin/env sh

# Install proto3 from source
#  brew install autoconf automake libtool
#  git clone https://github.com/google/protobuf
#  ./autogen.sh ; ./configure ; make ; make install
#
# Update protoc Go bindings via
#  go get -u github.com/golang/protobuf/{proto,protoc-gen-go}
#
# See also
#  https://github.com/grpc/grpc-go/tree/master/examples

protoc -I/usr/local/include -I. \
 -I.. \
 -I$GOPATH/src \
 --go_out=Mgoogle/api/annotations.proto=github.com/google/api,plugins=grpc:. \
 exchange_rate.proto service.proto
