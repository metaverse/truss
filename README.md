# truss

What even is the future?

![Everything all the time forever](http://i.imgur.com/FCmSUiQ.png)

## Install

To use this software, you must:
- Install the standard C++ implementation of protocol buffers from
	https://developers.google.com/protocol-buffers/
- Of course, install the Go compiler and tools from
	https://golang.org/
  See
	https://golang.org/doc/install
  for details or, if you are using gccgo, follow the instructions at
	https://golang.org/doc/install/gccgo
- Grab the code from the repository and install the proto package.
  The simplest way is to run `go get -u github.com/golang/protobuf/{proto,protoc-gen-go}`.
  The compiler plugin, protoc-gen-go, will be installed in $GOBIN,
  defaulting to $GOPATH/bin.  It must be in your $PATH for the protocol
  compiler, protoc, to find it.
- `$ go get -u google.golang.org/grpc`
- `$ go get -u github.com/TuneLab/gob/...`

## Docker

BETA

To build the docker image
`$ docker build -t tunelab/gob/truss .`

To use the docker image as `truss` on .proto files
`$ docker run -it --rm --name test -v $PWD:/gopath/src/microservice -w /gopath/src/microservice tunelab/gob/truss *.proto`


