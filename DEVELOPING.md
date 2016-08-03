# How to develop truss

## Dependencies

1. Everything required to install `truss`
2. go-bindata for compiling templates into binary `$ go get github.com/jteeuwen/go-bindata`

## Building

Whenever templates are modified, the templates must be recompiled to binary, this is done with:

```
$ go generate github.com/TuneLab/gob/...
```

Then to build truss and it's two protoc plugins to your $GOPATH/bin directory:

```
$ go install github.com/TuneLab/gob/...
```

## Testing

Before submitting a pull request always run tests that cover modified code. Also build truss and run truss's integration test. This can be done by

```
$ cd $GOPATH/src/github.com/TuneLab/gob/truss
$ make clean && make && make test
# If the tests failed and you want to remove generated code
$ make testclean
```

## Structure

Truss is composed of several libraries and programs which work in tandem. Here
are the main things to know about the internals of this project.

- `truss` is the program which unites the functionality of all other components in this project, spending most of it's time executing other programs. It's source lives in the `truss/` directory.
- `protoc-gen-truss-gokit` is a program and `protoc` plugin. It is responsible for creating and managing the files which make up your microservice. It's source lives in the `protoc-gen-truss-gokit/` directory.
- `protoc-gen-truss-doc` is a program and `protoc` plugin. It is responsible for creating documentation from the protobuf definition. It's source lives in the `protoc-gen-truss-doc/` directory.

Additional internal packages of note used by these programs are:

- `astmodifier`, located in `protoc-gen-truss-gokit/astmodifier/`, used to modify go files in place, and used by `protoc-gen-truss-gokit`
- `doctree`, located in `gendoc/doctree/`, which makes sense of the protobuf file passed to it by `protoc`, and is used by `protoc-gen-truss-gokit` and `protoc-gen-truss-doc`

## Docker

BETA

To build the docker image
`$ docker build -t tunelab/gob/truss .`

To use the docker image as `truss` on .proto files
`$ docker run -it --rm --name test -v $PWD:/gopath/src/microservice -w /gopath/src/microservice tunelab/gob/truss *.proto`
