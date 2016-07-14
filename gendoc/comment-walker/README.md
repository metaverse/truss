
# Comment Walker


This is a basic demo of parsing the comment data of a protobuf file within a protoc plugin written in go.

## How do we "walk comments"?

To understand how comments are traversed, you will probably want to start with [this explanation from the docs on how `protoc-gen-go` associates comments with their surrounding code declarations.](https://godoc.org/github.com/golang/protobuf/protoc-gen-go/descriptor#SourceCodeInfo_Location) In short, each comment contains a "path" of integers which refers to the struct in the AST which is associated with that comment. To actually traverse these paths, one must use the Go `reflect` package to parse the struct tags for each struct one is considering, since the numbers in the path refer to the message numbers in the protobuf files. Since those message numbers don't have a direct analog in Go, they're stored in the struct tags of each field of their cooresponding Go structs. The only way to access these struct tags within Go is to use the `reflect` package.

