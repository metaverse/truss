# Truss [![Build Status](https://travis-ci.org/TuneLab/go-truss.svg?branch=master)](https://travis-ci.org/TuneLab/go-truss)

Truss handles the painful parts of microservices, freeing you to focus on the
business logic.

**Please note that Truss is currently pre-release software, and may change
drastically with no notice. There is no versioning, no guarantees, no stability
at this time. However, if you want to play around, make suggestions, or submit
changes, we welcome issues and pull requests!**

![Everything all the time forever](http://i.imgur.com/FCmSUiQ.png)

## Install

Currently, there is no binary distribution of Truss, you must install from source.

To install this software, you must:

1. Install the standard C++ implementation of protocol buffers from
   https://developers.google.com/protocol-buffers/
2. Of course, install the Go compiler and tools from https://golang.org/. See
   https://golang.org/doc/install for details or, if you are using gccgo,
   follow the instructions at https://golang.org/doc/install/gccgo
4. Install the `protoc-gen-go` and `proto` packages for Go. The simplest way is
   to run `go get -u github.com/golang/protobuf/{proto,protoc-gen-go}`. The
   compiler plugin, protoc-gen-go, will be installed in `$GOBIN`, defaulting to
   `$GOPATH/bin`.  It must be in your `$PATH` for the protocol compiler,
   protoc, to find it.
5. Install gRPC: `go get -u google.golang.org/grpc`
6. Install Truss with `go get -u github.com/TuneLab/go-truss/...`, or
   `go install github.com/TuneLab/go-truss/...` if you're building locally.

## Usage

Using Truss is easy. Define your microservice in a protobuf file, and Truss
will use it to create an entire microservice.

Once you've written the definition of your microservice, which for example purposes
here is contained in a file called `my-service.proto`, invoke Truss on it:
```
$ truss my-service.proto
```
This will generate file in a directory called `NAME-service/` within
the current directory, where NAME is the package name as defined in the
protobuf service definition file.

See [USAGE.md](./USAGE.md) for more information.

## Developing

See [DEVELOPING.md](./DEVELOPING.md) for details.

<!--
TODO: Add example here of proto file, and the steps to create a microservice from it.
-->

