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
6. Install Truss with `$ go get -u github.com/TuneLab/gob/...`


## Docker

BETA

To build the docker image
`$ docker build -t tunelab/gob/truss .`

To use the docker image as `truss` on .proto files
`$ docker run -it --rm --name test -v $PWD:/gopath/src/microservice -w /gopath/src/microservice tunelab/gob/truss *.proto`

## Usage

Using Truss is easy. You define your microservice in a protobuf file, and Truss
uses that definition to create an entire microservice.

Once you've written the definition of your microservice, use the command `$ truss
{NAME_OF_PROTO_FILE}` to generate your microservice into a directory called
`service/` within your current directory.

<!--
TODO: Add example here of proto file, and the steps to create a microservice from it.
   -->

## Structure

Truss is composed of several libraries and programs which work in tandem. Here
are the main things to know about the internals of this project.

- `truss` is the program which unites the functionality of all other components in this project, spending most of it's time executing other programs. It's source lives in the `truss/` directory.
- `protoc-gen-truss-gokit` is a program and `protoc` plugin. It is responsible for creating and managing the files which make up your microservice. It's source lives in the `protoc-gen-truss-gokit/` directory.
- `protoc-gen-gendoc` is a program and `protoc` plugin. It is responsible for creating documentation from the protobuf definition. It's source lives in the `gendoc/` directory.

Additional internal packages of note used by these programs are:

- `astmodifier`, located in `protoc-gen-truss-gokit/astmodifier/`, used to modify go files in place, and used by `protoc-gen-truss-gokit`
- `doctree`, located in `gendoc/doctree/`, which makes sense of the protobuf file passed to it by `protoc`, and is used by `protoc-gen-truss-gokit` and `protoc-gen-gendoc`


