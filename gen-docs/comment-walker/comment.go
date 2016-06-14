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

// Parses a protobuf string to return the label of the field, if it exists.
func protoFieldLabel(proto_tag string) string {
	comma_split := strings.Split(proto_tag, ",")
	if len(comma_split) > 3 {
		eq_split := strings.Split(comma_split[3], "=")
		if len(eq_split) > 1 {
			return eq_split[1]
		}
	}
	return ""
}

// Returns the requested protobuf message field
func getProtobufField(proto_field int, proto_msg reflect.Value, depth int) (reflect.Value, string, error) {
	// 'marker' used as the representation of the input value for this
	// function, helps keep things consistent in logs
	//marker := proto_msg.Type().String()
	//logfd(depth-1, "Getting the protobuf field '%v' from '%v'\n", proto_field, marker)

	// if this message/field isn't a struct, then it's got to be an
	// array-indexable collection (by convention of protoc)
	if proto_msg.Kind() != reflect.Struct {
		marker := proto_msg.Index(proto_field).Type().String()
		logfd(depth, "The passed in value to search is a not a struct, assuming it is an indexable type.\n")
		logfd(depth, "Index: %v %v\n", proto_field, marker)
		return proto_msg.Index(proto_field), string(proto_field), nil
	}
	// Iterate through the fields of the struct, finding the field with the
	// struct tag indicating the protobuf field we're looking for.
	for n := 0; n < proto_msg.Type().NumField(); n++ {
		var typeField reflect.StructField = proto_msg.Type().Field(n)

		// Get the protobuf field number from the tag and check if it matches
		// the one we're looking for.
		pfield_n := -1
		tag := typeField.Tag.Get("protobuf")
		field_label := protoFieldLabel(tag)

		if len(tag) != 0 {
			pfield_n, _ = strconv.Atoi(strings.Split(tag, ",")[1])
		}
		if pfield_n != -1 && pfield_n == proto_field {
			// Found the correct field, return it and it's label
			logfd(depth, "Field '%02d, %02d' named '%v' is correct\n", n, pfield_n, field_label)
			return proto_msg.Field(n), field_label, nil
		} else {
			logfd(depth, "Field '%02d, %02d' named '%v' is NOT the correct field\n", n, pfield_n, field_label)
		}
	}
	// Couldn't find a proto field with the given index
	return proto_msg, "", fmt.Errorf("Couldn't find a proto field with the given index '%v'", proto_field)
}

func getCollectionIndex(node reflect.Value, index int) reflect.Value {
	return node.Index(index)
}

func walkNextStruct(path []int32, node reflect.Value, depth int) {
	if node.Kind() != reflect.Struct {
		panic("Walk next struct can only take a value of a struct!")
	}
	st_name := *node.FieldByName("Name").Interface().(*string)

	// Derive special information about this location, since it is the terminus
	// of the path
	if len(path) == 0 {
		logfd(depth, "Name of terminus struct: '%v'\n\n", st_name)
		return
	}
	logfd(depth, "Name of current struct: '%v'\n", st_name)

	// Field will almost definitely point to an array
	field, _, err := getProtobufField(int(path[0]), node, depth+1)
	if err != nil {
		panic(err)
	}

	next_node := getCollectionIndex(field, int(path[1]))

	// Dereference the returned field, if it exists
	var clean_next reflect.Value
	if next_node.Kind() == reflect.Ptr {
		clean_next = next_node.Elem()
	} else {
		clean_next = next_node
	}

	walkNextStruct(path[2:], clean_next, depth+1)
}

// Walk the given path, from the root file descriptor to the field/index which
// is the destination of the path.
func walkPath(path []int32, node interface{}, depth int) {

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
		logfd(depth, "Name of method: %v\n\n", *elm.FieldByName("Name").Interface().(*string))
		return
	}

	logfd(depth, "Path: %v\n", path)

	// Get the field for this portion of the path
	temp_field, _, _ := getProtobufField(int(path[0]), elm, depth+1)

	logfd(depth, "Field for path %v: %v\n", path[0], temp_field.Type())

	// Recurse!
	walkPath(path[1:], temp_field.Interface(), depth+1)

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

	depth := 0

	// Print a ton of fields
	for _, name := range request.FileToGenerate {
		logf("File to generate: %v\n", name)
	}
	for _, file := range request.GetProtoFile() {
		logfd(depth, "Proto file: %v\n", file.GetName())

		for _, msg := range file.MessageType {
			logfd(depth+1, "Msg: %v\n", msg.GetName())
		}

		for _, srvc := range file.Service {
			logfd(depth+1, "Service: %v\n", srvc.GetName())
			for _, meth := range srvc.GetMethod() {
				logfd(depth+2, "Method: %v\n", meth.GetName())
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
				logfd(depth+1, "Leading Comments: '%v' %v\n", strings.TrimSpace(lead), location.Path)
				// Print path information for this source code location
				walkNextStruct(location.Path, reflect.ValueOf(*file), depth+1)
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
