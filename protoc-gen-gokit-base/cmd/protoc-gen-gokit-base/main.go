package main

import (
	"io"
	"io/ioutil"
	"os"

	generator "github.com/TuneLab/gob/protoc-gen-gokit-base/generator"
	"github.com/TuneLab/gob/protoc-gen-gokit-base/util"
	"github.com/gengo/grpc-gateway/protoc-gen-grpc-gateway/descriptor"
	"github.com/golang/protobuf/proto"
	_ "github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
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

	registry := descriptor.NewRegistry()
	request, err := parseReq(os.Stdin)

	if err := registry.Load(request); err != nil {
		return
	}

	var targets []*descriptor.File
	for _, target := range request.FileToGenerate {
		util.Logf("file to be processed: %v\n", target)
		f, err := registry.LookupFile(target)
		_ = err
		targets = append(targets, f)
	}

	g := generator.New(registry, targets)

	codeGenFiles, _ := g.GenerateResponseFiles(targets)

	output := &plugin.CodeGeneratorResponse{
		File: codeGenFiles,
	}

	buf, err := proto.Marshal(output)
	_ = err

	if _, err := os.Stdout.Write(buf); err != nil {
		util.Logf("%v\n", err)
	}
}
