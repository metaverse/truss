# truss


The `truss` binary reads in gRPC files that define a single gRPC *service* and outputs:

1. Markdown and html documentation based on comments in the gRPC files.
2. Golang code for a gokit microservice that includes:
	- Logging
	- Metrics/Instrumentation
	- gRPC transport
	- http/json transport (including all encoding/decoding)
	- no-op handler methods for each *service* rpc, ready for business logic to be added
3. Golang code for a cli gokit microservice client that includes:
	- gRPC transport
	- http/json transport (including all encoding/decoding)
	- no-op handler methods for each *service* rpc, ready for marshalling command line arguments into a request object
4. An web based api explorer (through naive swagger generation)

## Install

```
$ go get github.com/TuneLab/gob/truss/cmd/...
$ go install github.com/TuneLab/gob/truss/cmd/...
$ truss microservice.md
```

## Requirements

`truss` must:
- Be invoked from some directory within your `$GOPATH/src`
- Be passed `.proto` file paths that:
	- Are withing the current directory

## Implementation details

### Generated file structure

We invoke `$ truss` in
```
.
└─
```
Lets say we have a `microservice.proto` file. With the defined *service* named `foobar`.
```
.
└── microservice.proto
```
We invoke `$ truss` from `.`  
`$ truss microservice.proto`  
  
Five stages of generation happen.

1. The `service` directory is made and gRPC `google.api.http` annotation dependencies are created

	```
	.
	├── service
	│   └── DONOTEDIT
	│       └── third_party
	│           └── googleapis
	│               └── google
	│                   └── api
	│                       ├── annotations.pb.go
	│                       ├── annotations.proto
	│                       ├── http.pb.go
	│                       └── http.proto
	└─── microservice.proto
	```

2. The `microservice.proto` file is parsed by the grpc go_out plugin for protoc which generated golang code for grpc communication as well as interfaces and structs for the *service* and all *messages*.

	```
	$ PWD=$(pwd)
	$ TRUSSIMPORT=${pwd#$GOPATH/src/}
	$ TRUSSGOOGLEAPI=foobar/DONOTEDIT/third_party/googleapis

	$ protoc -I/usr/local/include -I. \
		-I$pwd/$TRUSSGOOGLEAPI \
		--go_out=Mgoogle/api/annotations.proto=$TRUSSIMPORT/$TRUSSGOOGLEAPI/google/api, \
		plugins=grpc:./foobar/DONOTEDIT/compiledpb \
		microservice.proto

	```
	Which gives us the directory structure
	```
	.
	├── service
	│   └── DONOTEDIT
	│       ├── compiledpb
	│       │   └── microservice.pb.go
	│       └── third_party
	│           └── ...
	└── microservice.proto
	```

3. The `microservice.proto` file is parsed by the documentation generator which generated Markdown and html documentation for the *service* and all *messages*

	```
	$ protoc -I/usr/local/include -I. \
		-I$pwd/$TRUSSGOOGLEAPI \
		--truss_gendoc_out=./service/docs \
		microservice.proto
	```
	```
	.
	├── service
	│   ├── docs
	│   │   ├── docs.html
	│   │   └── docs.md
	│   └── DONOTEDIT
	│       ├── compiledpb
	│       │   └── ...
	│       └── third_party
	│           └── ...
	└── service.proto
	```

4. The `microservice.proto` file is parsed by the service/client generator which generates the golang server and client implementation of the *service*

	```
	$ protoc -I/usr/local/include -I. \
		-I$pwd/$TRUSSGOOGLEAPI \
		--truss_gokit_out=./service/DONOTEDIT \
		microservice.proto
	```
	```
	.
	├── service
	│   ├── client
	│   │   └── clienthandler.go
	│   ├── docs
	│   │   └── ...
	│   ├── server
	│   │   └── servicehandler.go
	│   └── DONOTEDIT
	│       ├── client
	│       │   ├── grpc
	│       │   │   └── client.go
	│       │   └── http
	│       │       └── client.go
	│       ├── cmd
	│       │   ├── cliclient
	│       │   │   └── main.go
	│       │   └── svc
	│       │       └── main.go
	│       ├── compiledpb
	│       │   └── ...
	│       ├── doc.go
	│       ├── endpoints.go
	│       ├── third_party
	│       │   └── ...
	│       ├── transport_grpc.go
	│       └── transport_http.go
	└── microservice.proto
	```

5. Finally, the services are built

	```
	.
	├── service
	│   ├── bin
	│   │   ├── cliclient
	│   │   └── svc
	│   ├── client
	│   │   └── clienthandler.go
	│   ├── docs
	│   │   └── ...
	│   ├── server
	│   │   └── servicehandler.go
	│   └── DONOTEDIT
	│       ├── client
	│       │   ├── grpc
	│       │   │   └── client.go
	│       │   └── http
	│       │       └── client.go
	│       ├── cmd
	│       │   ├── cliclient
	│       │   │   └── main.go
	│       │   └── svc
	│       │       └── main.go
	│       ├── compiledpb
	│       │   └── ...
	│       ├── doc.go
	│       ├── endpoints.go
	│       ├── third_party
	│       │   └── ...
	│       ├── transport_grpc.go
	│       └── transport_http.go
	└── microservice.proto
	```

## TODO:

Provide errors for:
  - If an rpc has the name "NewBasicService"
  - If the *service* is named the same as any `directory` in the directory `$ truss` was invoked from.

