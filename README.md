# Truss [![Build Status](https://travis-ci.org/TuneLab/go-truss.svg?branch=master)](https://travis-ci.org/TuneLab/go-truss)

Truss handles the painful parts of services, freeing you to focus on the business logic.

**Please note that Truss is currently pre-release software, and may change drastically with no notice. There is no versioning, no guarantees, no stability at this time. However, if you want to play around, make suggestions, or submit changes, we welcome issues and pull requests!**

![Everything all the time forever](http://i.imgur.com/FCmSUiQ.png)

## Install

Currently, there is no binary distribution of Truss, you must install from source.

To install this software, you must:

1. Install protocol buffers version 3 or greater. The easiest way is to download a release from the [github releases](https://github.com/google/protobuf/releases) and add the binary to your `$PATH`. Otherwise [install from source.](https://github.com/google/protobuf/releases)
2. Of course, install the Go compiler and tools from https://golang.org/. See https://golang.org/doc/install for details.
3. Install the `protoc-gen-go` and `proto` packages for Go. The simplest way is to run 

	```
	go get -u github.com/golang/protobuf/{proto,protoc-gen-go}
	```

	The compiler plugin, protoc-gen-go, will be installed in `$GOPATH/bin`.  It must be in your `$PATH` for the protocol compiler, `protoc`, to find it.
4. Install Truss with 

	```
	go get -u github.com/TuneLab/go-truss/...
	```

## Usage

Using Truss is easy. You define your service with [gRPC](http://www.grpc.io/) and [protoc buffers](https://developers.google.com/protocol-buffers/docs/proto3), and Truss uses that definition to create an entire service. You can even add [http annotations](
https://github.com/googleapis/googleapis/blob/928a151b2f871b4239b7707e1bb59258df3fe10a/google/api/http.proto#L36) for HTTP 1.1/JSON transport!

Then you open the `handlers/server/server_handler.go`, add you business logic, and you're good to go.

Here is an example service definition: [Echo Service](./example/echo.proto)

Try Truss for yourself on Echo Service to see the service that is generated:

```
truss example/echo.proto
```

See [USAGE.md](./USAGE.md) for more details.

## Developing

See [DEVELOPING.md](./DEVELOPING.md) for details.
