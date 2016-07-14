package main

import (
	"flag"
	"io"
	"io/ioutil"
	"os"

	"github.com/TuneLab/gob/gendoc/doctree"
	"github.com/TuneLab/gob/gendoc/doctree/makedt"
	"github.com/golang/protobuf/proto"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

var (
	response = string("")
)

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

// Returns name of the output directory for the 'docs.md' file. For now, is the
// name of the only service in the given package.
func outputDir(dt doctree.Doctree) string {
	md := dt.(*doctree.MicroserviceDefinition)
	svc_name := ""
	for _, file := range md.Files {
		for _, svc := range file.Services {
			svc_name = svc.GetName()
		}
	}
	return svc_name
}

func main() {
	flag.Parse()

	request, err := parseReq(os.Stdin)
	if err != nil {
		panic(err)
	}

	// Parse the proto files we've been given, then create the markdown
	// documentation for those proto files. All the documentation is written to
	// a file named 'docs.md'.
	doc, _ := makedt.New(request)
	response := doc.Markdown()

	out_fname := outputDir(doc) + "/docs.md"
	response_file := str_to_response(response, out_fname)
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
