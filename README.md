# Truss [![Build Status](https://travis-ci.org/TuneLab/go-truss.svg?branch=master)](https://travis-ci.org/TuneLab/go-truss)

Truss handles the painful parts of services, freeing you to focus on the
business logic.

**Please note that Truss is currently pre-release software, and may change
drastically with no notice. There is no versioning, no guarantees, no stability
at this time. However, if you want to play around, make suggestions, or submit
changes, we welcome issues and pull requests!**

![Everything all the time forever](http://i.imgur.com/FCmSUiQ.png)

## Install

Currently, there is no binary distribution of Truss, you must install from
source.

To install this software, you must:

1. Install protoc 3 or newer. The easiest way is to
download a release from [github](https://github.com/google/protobuf/releases)
and add to `$PATH`.
Otherwise [install from source.](https://github.com/google/protobuf)
1. Install the `protoc-gen-go` package (must be in `$PATH`).

	```
	go get -u github.com/golang/protobuf/protoc-gen-go
	```
1. Install HTTP Annotations

	```
	go get -u github.com/TuneLab/go-genproto
	```
1. Install Truss with

	```
	go get -u github.com/TuneLab/go-truss/...
	```

## Usage

Using Truss is easy. You define your service with [gRPC](http://www.grpc.io/)
and [protoc buffers](https://developers.google.com/protocol-buffers/docs/proto3),
and Truss uses that definition to create an entire service. You can even
add [http annotations](
https://github.com/googleapis/googleapis/blob/928a151b2f871b4239b7707e1bb59258df3fe10a/google/api/http.proto#L36)
for HTTP 1.1/JSON transport!

Then you open the `handlers/server/server_handler.go`,
add you business logic, and you're good to go.

Here is an example service definition: [Echo Service](./_example/echo.proto)

Try Truss for yourself on Echo Service to see the service that is generated:

```
truss _example/echo.proto
```

See [USAGE.md](./USAGE.md) and [TUTORIAL.md](./TUTORIAL.md) for more details.

## Developing

See [DEVELOPING.md](./DEVELOPING.md) for details.
