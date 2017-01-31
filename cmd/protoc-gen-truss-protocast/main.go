package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/golang/protobuf/proto"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

func main() {
	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	req := new(plugin.CodeGeneratorRequest)
	if err = proto.Unmarshal(input, req); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	filesOut := req.GetFileToGenerate()
	//fmt.Fprintln(os.Stderr, "test")

	fileName := filesOut[0]
	protocOut := string(input)

	codeGenFile := plugin.CodeGeneratorResponse_File{
		Name:    &fileName,
		Content: &protocOut,
	}

	output := &plugin.CodeGeneratorResponse{
		File: []*plugin.CodeGeneratorResponse_File{
			&codeGenFile,
		},
	}

	buf, err := proto.Marshal(output)

	if _, err := os.Stdout.Write(buf); err != nil {
		os.Exit(1)
	}
}
