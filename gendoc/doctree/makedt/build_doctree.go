package makedt

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/TuneLab/gob/gendoc/doctree"
	"github.com/TuneLab/gob/gendoc/doctree/httpopts"
	"github.com/TuneLab/gob/gendoc/svcparse"

	log "github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

func init() {
	// Output to stderr instead of stdout, could also be a file.
	log.SetOutput(os.Stderr)
	// Force colors in logs to be on
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})

	// Only log the warning severity or above.
	log.SetLevel(log.InfoLevel)
}

// Finds the package name of the proto files named on the command line
func findDoctreePackage(req *plugin.CodeGeneratorRequest) string {
	for _, cmd_file := range req.GetFileToGenerate() {
		for _, proto_file := range req.GetProtoFile() {
			if proto_file.GetName() == cmd_file {
				return proto_file.GetPackage()
			}
		}
	}
	return ""
}

// Finds a message given a fully qualified name to that message.
func findMessage(md *doctree.MicroserviceDefinition, new_file *doctree.ProtoFile, path string) (*doctree.ProtoMessage, error) {
	if path[0] == '.' {
		parts := strings.Split(path, ".")
		for _, file := range md.Files {
			for _, msg := range file.Messages {
				if parts[2] == msg.GetName() {
					return msg, nil
				}
			}
		}
		for _, msg := range new_file.Messages {
			if parts[2] == msg.GetName() {
				return msg, nil
			}
		}
	} else {
		for _, msg := range new_file.Messages {
			if path == msg.GetName() {
				return msg, nil
			}
		}
	}
	return nil, fmt.Errorf("Couldn't find message.")

}

// New accepts a Protobuf CodeGeneratorRequest and returns a Doctree struct
func New(req *plugin.CodeGeneratorRequest) (doctree.Doctree, error) {
	dt := doctree.MicroserviceDefinition{}
	dt.SetName(findDoctreePackage(req))
	for _, file := range req.ProtoFile {
		// Check if this file is one we even should examine, and if it's not,
		// skip it
		if file.GetPackage() != findDoctreePackage(req) {
			continue
		}

		// This is a file we are meant to examine, so contine with it's
		// creation in the Doctree
		new_file := doctree.ProtoFile{}
		new_file.Name = file.GetName()

		// Add enums to this file
		for _, enum := range file.EnumType {
			new_enum := doctree.ProtoEnum{}
			new_enum.SetName(enum.GetName())
			for _, val := range enum.GetValue() {
				// Add values to this enum
				n_val := doctree.EnumValue{}
				n_val.SetName(val.GetName())
				n_val.Number = int(val.GetNumber())
				new_enum.Values = append(new_enum.Values, &n_val)
			}
			new_file.Enums = append(new_file.Enums, &new_enum)
		}

		// Add messages to this file
		for _, msg := range file.MessageType {
			new_msg := doctree.ProtoMessage{}
			new_msg.Name = *msg.Name
			// Add fields to this message
			for _, field := range msg.Field {
				new_field := doctree.MessageField{}
				new_field.Number = int(field.GetNumber())
				new_field.Name = *field.Name
				new_field.Type.Name = field.GetTypeName()
				// The `GetTypeName` method on FieldDescriptorProto only
				// returns the path/name of a type if that type is a message or
				// an Enum. For basic types (int, float, etc.) `GetTypeName()`
				// returns an empty string. In that case, we set the new_fields
				// type name to be the string representing the type of the
				// field being examined.
				if new_field.Type.Name == "" {
					new_field.Type.Name = field.Type.String()
				}
				// The label we get back is a number, translate it to a human
				// readable string
				label := int32(field.GetLabel())
				label_name := descriptor.FieldDescriptorProto_Label_name[label]
				new_field.Label = label_name

				new_msg.Fields = append(new_msg.Fields, &new_field)
			}
			new_file.Messages = append(new_file.Messages, &new_msg)
		}

		// Add services to this file
		for _, srvc := range file.Service {
			n_svc := doctree.ProtoService{}
			n_svc.Name = *srvc.Name
			n_svc.FullyQualifiedName = "." + file.GetPackage() + "." + n_svc.Name

			// Add methods to this service
			for _, meth := range srvc.Method {
				n_meth := doctree.ServiceMethod{}
				n_meth.Name = *meth.Name

				// Set this methods request and responses to point to existing
				// Message types
				req_msg, err := findMessage(&dt, &new_file, *meth.InputType)
				if req_msg == nil || err != nil {
					panic(fmt.Sprintf("Couldn't find message type for '%v'\n", *meth.InputType))
				}
				resp_msg, err := findMessage(&dt, &new_file, *meth.OutputType)
				if resp_msg == nil || err != nil {
					panic(fmt.Sprintf("Couldn't find message type for '%v'\n", *meth.OutputType))
				}
				n_meth.RequestType = req_msg
				n_meth.ResponseType = resp_msg

				n_svc.Methods = append(n_svc.Methods, &n_meth)
			}

			new_file.Services = append(new_file.Services, &n_svc)
		}
		dt.Files = append(dt.Files, &new_file)
	}

	// Do the association of comments to units code. The implementation of this
	// function is in `associate_comments.go`
	doctree.AssociateComments(&dt, req)

	addHttpOptions(&dt, req)

	return &dt, nil
}

// Searches all descendent directories for a file with name `fname`.
func searchFileName(fname string) string {
	fname = path.Base(fname)
	foundPath := ""
	visitor := func(path string, info os.FileInfo, err error) error {
		if info.Name() == fname {
			foundPath = path
		}
		return nil
	}
	_ = filepath.Walk("./", visitor)
	return foundPath
}

// Parse the protobuf files for comments surrounding http options, then add
// those to the Doctree in place.
func addHttpOptions(dt doctree.Doctree, req *plugin.CodeGeneratorRequest) {

	fname := FindServiceFile(req)
	full_path := searchFileName(fname)

	f, err := os.Open(full_path)
	if err != nil {
		cwd, _ := os.Getwd()
		log.Warnf("From current directory '%v', error opening file '%v', '%v'\n", cwd, full_path, err)
		log.Warnf("Due to the above warning(s), http options and bindings where not parsed and will not be present in the generated documentation.")
		return
	}
	lex := svcparse.NewSvcLexer(f)
	parsed_svc, err := svcparse.ParseService(lex)

	if err != nil {
		log.Warnf("Error found while parsing file '%v': %v", full_path, err)
		log.Warnf("Due to the above warning(s), http options and bindings where not parsed and will not be present in the generated documentation.")
		return
	}

	svc := dt.GetByName(fname).GetByName(parsed_svc.GetName()).(*doctree.ProtoService)
	for _, pmeth := range parsed_svc.Methods {
		meth := svc.GetByName(pmeth.GetName()).(*doctree.ServiceMethod)
		meth.HttpBindings = pmeth.HttpBindings
	}

	// Assemble the http parameters for each http binding
	httpopts.Assemble(dt)
}

// Searches through the files in the request and returns the path to the first
// one which contains a service declaration. If no file in the request contains
// a service, returns an empty string.
func FindServiceFile(req *plugin.CodeGeneratorRequest) string {
	svc_files := []string{}
	// Since the names of proto files in FileDescriptorProto's don't contain
	// the path, we have to find the first one with a service, then find it's
	// actual relative path by searching the slice `FileToGenerate`.
	for _, file := range req.GetProtoFile() {
		if len(file.GetService()) > 0 {
			svc_files = append(svc_files, file.GetName())
		}
	}
	for _, file := range req.GetFileToGenerate() {
		for _, svc_f := range svc_files {
			if strings.Contains(file, svc_f) {
				return file
			}
		}
	}
	return ""
}
