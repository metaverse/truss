# Truss

Truss handles the painful parts of microservices, freeing you to focus on the business logic.

![Everything all the time forever](http://i.imgur.com/FCmSUiQ.png)

## Install

Currently, there is no binary distribution of Truss, you must install from source.

To install this software, you must:

1. Install the standard C++ implementation of protocol buffers from https://developers.google.com/protocol-buffers/
2. Of course, install the Go compiler and tools from https://golang.org/. See https://golang.org/doc/install for details or, if you are using gccgo, follow the instructions at https://golang.org/doc/install/gccgo
4. Install the `protoc-gen-go` and `proto` packages for Go. The simplest way is to run `go get -u github.com/golang/protobuf/{proto,protoc-gen-go}`. The compiler plugin, protoc-gen-go, will be installed in `$GOBIN`, defaulting to `$GOPATH/bin`.  It must be in your `$PATH` for the protocol compiler, protoc, to find it.
5. Install the gRPC: `$ go get -u google.golang.org/grpc`
6. Install Truss with `$ go get -u github.com/TuneLab/go-truss/...`

## Usage

Using Truss is easy. You define your microservice in a protobuf file, and Truss
uses that definition to create an entire microservice.

Once you've written the definition of your microservice, use the command `$ truss
{NAME_OF_PROTO_FILE}` to generate your microservice into a directory called
`service/` within your current directory.

## Developing

See [DEVELOPING.md](./DEVELOPING.md) for details

<!--
TODO: Add example here of proto file, and the steps to create a microservice from it.
   -->

