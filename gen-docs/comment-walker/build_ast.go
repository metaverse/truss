package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/TuneLab/gob/gen-docs/doctree"
	"github.com/golang/protobuf/proto"
	descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

var (
	_        = descriptor.MethodDescriptorProto{}
	response = string("")
	indent   = string("    ")
)

// A logging utility function
func logf(format string, args ...interface{}) {
	response += fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, format, args...)
}

// Logging utility function printing indentation of the specified depth
func logfd(depth int, format string, args ...interface{}) {
	for i := 0; i < depth; i++ {
		logf(indent)
	}
	logf(format, args...)
}

// Attempt to parse the incoming CodeGeneratorRequest being written by `protoc` to our stdin
func parseReq(r io.Reader) (*plugin.CodeGeneratorRequest, error) {
	input, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	req := new(plugin.CodeGeneratorRequest)
	if err = proto.Unmarshal(input, req); err != nil {
		return nil, err
	}
	return req, nil
}

func main() {
	flag.Parse()

	request, err := parseReq(os.Stdin)
	if err != nil {
		panic(err)
	}

	doc, _ := doctree.New(request)

	response := doc.String()
	response_file := str_to_response(response, "ast.log")
	output_struct := &plugin.CodeGeneratorResponse{File: []*plugin.CodeGeneratorResponse_File{response_file}}

	buf, err := proto.Marshal(output_struct)

	if _, err := os.Stdout.Write(buf); err != nil {
		panic(err)
	}
}

func str_to_response(instr string, fname string) *plugin.CodeGeneratorResponse_File {
	return &plugin.CodeGeneratorResponse_File{
		Name:    &fname,
		Content: &instr,
	}
}
