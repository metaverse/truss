
# `protoc-gen-truss-docs`

A `protoc` plugin which can generate documentation from an annotated Protobuf definition file. Handles http-options.

To run, ensure the program is installed by running `go install github.com/TuneLab/gob/protoc-gen-truss-doc/...`. Once installed, you can use this plugin by compiling a proto file with `protoc` and the the following options:

	protoc -I/usr/local/include -I. -I.. \
		-I$GOPATH/src/github.com/TuneLab/gob/third_party/googleapis/ \
		--truss-doc_out=. {NAME_OF_PROTO_FILE}

This will output a file in the current directory named "docs.md" containing a markdown representation of your documentation.


## Limitations and Bugs

Currently, there are a variety of limitations in the documentation parser.

- Having additional http bindings via the `additional_bindings` directive when declaring http options causes the parser to break.
