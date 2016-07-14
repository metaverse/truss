package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/golang/protobuf/proto"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"

	"github.com/TuneLab/gob/gendoc/doctree"
	"github.com/TuneLab/gob/gendoc/doctree/makedt"
	generator "github.com/TuneLab/gob/protoc-gen-truss-gokit/generator"
)

// parseReq reads io.Reader r into memory and attempts to marshal
// that input into a protobuf plugin CodeGeneratorRequest
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
	request, err := parseReq(os.Stdin)

	prototree, _ := makedt.New(request)
	outputDirName := outputDir(prototree)

	prototreeDefinition := prototree.(*doctree.MicroserviceDefinition)

	g := generator.New(prototreeDefinition.Files, outputDirName)

	codeGenFiles, _ := g.GenerateResponseFiles()

	output := &plugin.CodeGeneratorResponse{
		File: codeGenFiles,
	}

	buf, err := proto.Marshal(output)
	_ = err

	if _, err := os.Stdout.Write(buf); err != nil {
		os.Exit(1)
	}
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
