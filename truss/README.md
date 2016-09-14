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
$ go get -u -v github.com/TuneLab/go-truss/...

```

## Requirements

`truss` must:
- Be invoked from some directory within your `$GOPATH/src`
- Be passed `.proto` file paths that:
	- Are withing the current directory

