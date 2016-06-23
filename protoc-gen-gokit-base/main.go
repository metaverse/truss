package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	_ "github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

// parseReq reads io.Reader r into memory and attempts to marshal
// that input into a protobuf plugin CodeGeneratorRequest
func parseReq(r io.Reader) (*plugin.CodeGeneratorRequest, error) {
	glog.V(1).Info("Parsing code generator request")
	input, err := ioutil.ReadAll(r)
	if err != nil {
		glog.Errorf("Failed to read code generator request: %v", err)
		return nil, err
	}
	req := new(plugin.CodeGeneratorRequest)
	if err = proto.Unmarshal(input, req); err != nil {
		glog.Errorf("Failed to unmarshal code generator request: %v", err)
		return nil, err
	}
	glog.V(1).Info("Parsed code generator request")
	return req, nil
}

type empty struct{}

var (
	response = string("")
)

func logf(format string, args ...interface{}) {
	response += fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, format, args...)
}

func main() {

	defer glog.Flush()
	glog.V(1).Info("Processing code generator request")
	req, err := parseReq(os.Stdin)
	if err != nil {
		glog.Fatal(err)
	}

	glog.V(1).Info("Building Output")

	_ = req

	var codeGenFiles []*plugin.CodeGeneratorResponse_File
	for _, file := range AssetNames() {
		//logf("%v\n", paths)
		curResponseFile := plugin.CodeGeneratorResponse_File{}

		// Remove "template/" so that generated files do not include that directory
		d := strings.TrimPrefix(file, "template/")
		curResponseFile.Name = &d

		bytesOfFile, _ := Asset(file)

		stringFile := string(bytesOfFile)

		curResponseFile.Content = &stringFile
		codeGenFiles = append(codeGenFiles, &curResponseFile)
	}

	output := &plugin.CodeGeneratorResponse{
		File: codeGenFiles,
	}

	buf, err := proto.Marshal(output)
	if err != nil {
		glog.Fatal(err)
	}

	if _, err := os.Stdout.Write(buf); err != nil {
		glog.Fatal(err)
	}
}
