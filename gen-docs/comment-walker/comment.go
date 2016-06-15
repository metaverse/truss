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

	// Iterate through the fields of the struct, finding the field with the
	// struct tag indicating that that field correlates to the protobuf field
	// we're looking for.
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
			logfd(depth, "Field '%02d, %02d' labeled '%v' with type '%v' is correct\n", n, pfield_n, field_label, proto_msg.Field(n).Type())
			return proto_msg.Field(n), field_label, nil
		} else {
			logfd(depth, "Field '%02d, %02d' labeled '%v' is NOT the correct field\n", n, pfield_n, field_label)
		}
	}
	// Couldn't find a proto field with the given index
	return proto_msg, "", fmt.Errorf("Couldn't find a proto field with the given index '%v'", proto_field)
}

func getCollectionIndex(node reflect.Value, index int) reflect.Value {
	if index >= node.Len() {
		panic(fmt.Sprintf("The node '%v' is of length '%v', cannot access index '%v'", node, node.Len(), index))
	}
	return node.Index(index)
}

func walkNextStruct(path []int32, node reflect.Value, depth int) {
	var st_name string
	switch node.Kind() {
	case reflect.String:
		st_name = node.Interface().(string)
	case reflect.Ptr:
		node = node.Elem()
	default:
		if node.Kind() != reflect.Struct {
			panic(fmt.Sprintf("walkNextStruct expected struct, found '%v'", node.Kind()))
		} else {
			st_name = *node.FieldByName("Name").Interface().(*string)
		}
	}

	// Derive special information about this location, since it is the terminus
	// of the path
	if len(path) == 0 {
		logfd(depth, "Name of terminus struct: '%v'\n\n", st_name)
		return
	}
	logfd(depth, "Name of current struct: '%v' %v\n", st_name, path)

	field, field_label, err := getProtobufField(int(path[0]), node, depth+1)
	if err != nil {
		panic(err)
	}

	// If the path ends here, then the path is indicating this field, and not
	// anything more specific
	if len(path) == 1 {
		logfd(depth, "Label of terminus struct field: '%v'\n\n", field_label)
		return
	}

	// Since everything after this point is assuming that field is a slice, if
	// it's not we recurse
	if field.Kind() != reflect.Slice {
		walkNextStruct(path[1:], field, depth+1)
		return
	}

	if int(path[1]) >= field.Len() {
		logfd(depth, "WARNING: Encountered field '%v' with length '%v' not matching path '%v' currently being walked. Returning.", field_label, field.Len(), path)
		return
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
			logfd(depth+1, "Source location: %v\n", location.Path)
			lead := location.GetLeadingComments()
			if len(lead) > 1 {
				logfd(depth+2, "Leading Comments: '%v' %v\n", strings.TrimSpace(lead), location.Path)
			}
			logfd(depth+2, "Trailing comment: %v\n", location.GetTrailingComments())
			for _, v := range location.GetLeadingDetachedComments() {
				logfd(depth+2, "Leading detached comment: '%v\n'", strings.TrimSpace(v))
			}
			// Only walk the tree if this source code location has a comment
			// located with it. Not all source locations have valid paths, but
			// all sourcelocations with comments must point to concrete things,
			// so only recurse on those.
			if len(lead) > 1 || len(location.LeadingDetachedComments) > 1 {
				logfd(depth+1, "Begin walking tree for source location\n")
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
