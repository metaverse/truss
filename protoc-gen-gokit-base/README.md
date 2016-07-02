# Base Gokit Server Implementation

A `protoc` plugin that will build out a bare bones grpc service that builds out blank handlers for rpc services defined Protobuf definition files.


# NOTE:

This plugin uses the package `github.com/gengo/grpc-gateway/protoc-gen-grpc-gateway/descriptor`. Because of this, service rpc methods are only detected if they have an `option (google.api.http)`. In the style of an MVP, this will be dealt with later.
