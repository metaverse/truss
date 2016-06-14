package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

///////////////////////////////////////
//              What?
//
// This code is meant purely as a toy demo of some of how to approach
// navigating a Protobuf AST. This is an exploration of how to coorelate
// comment information in a Protobuf definition with the structs within. Right
// now it only finds this information, not coorelating these pieces in any way.

var (
	_        = descriptor.MethodDescriptorProto{}
	response = string("")
)

// Attempt to parse the incoming CodeGeneratorRequest being written by `protoc` to our stdin
func parseReq(r io.Reader) (*plugin.CodeGeneratorRequest, error) {
	glog.V(1).Info("Parsing code generator request")
	input, err := ioutil.ReadAll(r)
	if err != nil {
		glog.Errorf("Failed to read code generator request from stdin: %v", err)
		return nil, err
	}

	req := new(plugin.CodeGeneratorRequest)
	if err = proto.Unmarshal(input, req); err != nil {
		glog.Errorf("Failed to unmarshal code generator request: %v", err)
		return nil, err
	}
	glog.V(1).Info("Successfully parsed code generator request")
	return req, nil
}

// Returns the requested protobuf message field
func getProtobufField(proto_field int, proto_msg reflect.Value) (reflect.Value, int, error) {
	// if this message/field isn't a struct, then it's got to be an
	// array-indexable collection (by convention of protoc)
	if proto_msg.Kind() != reflect.Struct {
		response += fmt.Sprintf("\tField: %v %v\n", proto_field, proto_msg.Type())
		return proto_msg.Index(proto_field), proto_field, nil
	}
	// Iterate through the fields of the struct, finding the field with the
	// struct tag indicating the protobuf field we're looking for.
	for n := 0; n < proto_msg.Type().NumField(); n++ {
		var typeField reflect.StructField = proto_msg.Type().Field(n)

		// Get the protobuf field number from the tag and check if it matches
		// the one we're looking for.
		pfield_n := -1
		tag := typeField.Tag.Get("protobuf")

		response += fmt.Sprintf("Tag: %v\n", tag)

		if len(tag) != 0 {
			pfield_n, _ = strconv.Atoi(strings.Split(tag, ",")[1])
		}
		if pfield_n != -1 && pfield_n == proto_field {
			response += fmt.Sprintf("\tField: %v %v %v\n", n, proto_field, proto_msg.Type().Field(n).Name)
			return proto_msg.Field(n), n, nil
		}
	}
	// Couldn't find a proto field with the given index
	return proto_msg, -1, fmt.Errorf("Couldn't find a proto field with the given index '%v'", proto_field)
}

// Walk the given path, from the root file descriptor to the field/index which
// is the destination of the path.
func walkPath(path []int32, node interface{}) {

	//line := string("")

	// Iterate over the fields via reflection
	val := reflect.ValueOf(node)

	var elm reflect.Value

	// If the given node is a pointer, dereference that pointer, otherwise use
	// that value as our element
	if val.Kind() == reflect.Ptr {
		elm = val.Elem()
	} else {
		elm = val
	}

	// Derive special information about this location, since it is the terminus
	// of the path
	if len(path) == 0 {
		response += fmt.Sprintf("\tName of method: %v\n\n", *elm.FieldByName("Name").Interface().(*string))
		return
	}

	fmt.Fprintf(os.Stderr, "Path: %v\n", path)

	// Get the field for this portion of the path
	temp_field, _, _ := getProtobufField(int(path[0]), elm)

	response += fmt.Sprintf("\tField for path %v: %v\n", path[0], temp_field.Type())
	fmt.Fprintf(os.Stderr, "\tField for path %v: %v\n", path[0], temp_field.Type())

	// Recurse!
	walkPath(path[1:], temp_field.Interface())

	return

}

func main() {
	flag.Parse()
	defer glog.Flush()

	glog.V(1).Info("Processing the CodeGeneratorRequest")
	request, err := parseReq(os.Stdin)
	if err != nil {
		glog.Fatal(err)
	}

	// Print a ton of fields
	for _, name := range request.FileToGenerate {
		line := fmt.Sprintf("File to generate: %v\n", name)
		response += line
	}
	for _, file := range request.GetProtoFile() {
		line := fmt.Sprintf("Proto file: %v\n", file.GetName())
		response += line

		for _, msg := range file.MessageType {
			line = fmt.Sprintf("\tMsg: %v\n", msg.GetName())
			response += line
		}

		for _, srvc := range file.Service {
			line = fmt.Sprintf("\tService: %v\n", srvc.GetName())
			response += line
			for _, meth := range srvc.GetMethod() {
				line = fmt.Sprintf("\t\tMethod: %v\n", meth.GetName())
				response += line
			}
		}

		// Skip comments for files outside the main one being considered
		skip := true
		for _, gen := range request.FileToGenerate {
			if file.GetName() == gen {
				skip = false
			}
		}
		if skip {
			continue
		}

		// Print source code in the files
		info := file.GetSourceCodeInfo()
		for _, location := range info.GetLocation() {
			lead := location.GetLeadingComments()
			if len(lead) > 1 {
				line = fmt.Sprintf("\tLeading Comments: '%v' %v\n", lead, location.Path)
				response += line
				// Print path information for this source code location
				walkPath(location.Path, file)
			}
		}
	}

	// Create boilerplate response structs
	response_file := stringResponse(response)
	output_struct := &plugin.CodeGeneratorResponse{File: []*plugin.CodeGeneratorResponse_File{response_file}}

	buf, err := proto.Marshal(output_struct)

	if _, err := os.Stdout.Write(buf); err != nil {
		glog.Fatal(err)
	}
}

func stringResponse(instr string) *plugin.CodeGeneratorResponse_File {
	fname := string("result.log")
	return &plugin.CodeGeneratorResponse_File{
		Name:    &fname,
		Content: &instr,
	}
}
