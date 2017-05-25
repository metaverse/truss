# gengokit

1. Generates Golang code for a gokit microservice that includes:
	- Logging
	- Metrics/Instrumentation
	- gRPC transport
	- http/json transport (including all encoding/decoding)
	- no-op handler methods for each *service* rpc, ready for business logic to be added
2. Generates Golang code for a cli gokit microservice client that includes:
	- gRPC transport
	- http/json transport (including all encoding/decoding)
	- handler methods that marshal command line arguments into server requests
