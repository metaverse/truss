# truss

`truss` reads gRPC files that define a single gRPC service and outputs the following:

1. Markdown and HTML documentation based on comments in the gRPC files.
2. Golang code for a [Go Kit](http://gokit.io) microservice that includes the following:
	- gRPC transport
	- HTTP/JSON transport (including all encoding/decoding)
	- no-op handler methods for each service RPC, ready for business logic to be added
3. Golang code for a CLI Go Kit microservice client that includes the following:
	- gRPC transport
	- HTTP/JSON transport (including all encoding/decoding)
	- no-op handler methods for each service RPC, ready for marshalling command line
      arguments into a request object and sending a request to a server
4. An web based API explorer (through naive Swagger generation)

