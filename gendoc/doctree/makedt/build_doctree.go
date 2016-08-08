// Makedt is a package for exposing the creation of a doctree structure.
//
// It lives in it's own package because it must use several other packages
// which make use of doctree to create a doctree, so to prevent circular
// imports, it must be its own package.
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
	"github.com/pkg/errors"
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

// Finds a message given a fully qualified name to that message. The provided
// path may be either a fully qualfied name of a message, or just the bare name
// for a message.
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
		newFile, err := NewFile(file, &dt)
		if err != nil {
			return nil, errors.Wrapf(err, "file creation of '%s' failed", file.GetName())
		}
		dt.Files = append(dt.Files, newFile)
	}

	// Do the association of comments to units code. The implementation of this
	// function is in `associate_comments.go`
	doctree.AssociateComments(&dt, req)

	addHttpOptions(&dt, req)

	return &dt, nil
}

// Build a new doctree.File struct
func NewFile(
	pfile *descriptor.FileDescriptorProto,
	curNewDt *doctree.MicroserviceDefinition) (*doctree.ProtoFile, error) {

	newFile := doctree.ProtoFile{}
	newFile.Name = pfile.GetName()

	for _, enum := range pfile.EnumType {
		newEnum, err := NewEnum(enum)
		if err != nil {
			return nil, errors.Wrap(err, "error creating doctree.ProtoEnum")
		}
		newFile.Enums = append(newFile.Enums, newEnum)
	}

	for _, msg := range pfile.MessageType {
		newMsg, err := NewMessage(msg)
		if err != nil {
			return nil, errors.Wrap(err, "error creating doctree.ProtoMessage")
		}
		newFile.Messages = append(newFile.Messages, newMsg)
	}

	for _, srvc := range pfile.Service {
		newSvc, err := NewService(srvc, &newFile, curNewDt)
		if err != nil {
			return nil, errors.Wrap(err, "error creating doctree.ProtoService")
		}
		// Set the new services FullyQualifiedName here so that we don't have
		// to pass around additional references to pfile.
		newSvc.FullyQualifiedName = "." + pfile.GetPackage() + "." + newSvc.Name
		newFile.Services = append(newFile.Services, newSvc)
	}

	return &newFile, nil
}

// NewEnum returns a *doctree.ProtoEnum created from a *descriptor.EnumDescriptorProto
func NewEnum(enum *descriptor.EnumDescriptorProto) (*doctree.ProtoEnum, error) {
	newEnum := doctree.ProtoEnum{}

	newEnum.SetName(enum.GetName())
	// Add values to this enum
	for _, val := range enum.GetValue() {
		nval := doctree.EnumValue{}
		nval.SetName(val.GetName())
		nval.Number = int(val.GetNumber())
		newEnum.Values = append(newEnum.Values, &nval)
	}

	return &newEnum, nil
}

// NewMessage returns a *doctree.ProtoMessage created from a *descriptor.DescriptorProto
func NewMessage(msg *descriptor.DescriptorProto) (*doctree.ProtoMessage, error) {
	newMsg := doctree.ProtoMessage{}
	newMsg.Name = *msg.Name
	// Add fields to this message
	for _, field := range msg.Field {
		newField := doctree.MessageField{}
		newField.Number = int(field.GetNumber())
		newField.Name = *field.Name
		newField.Type.Name = field.GetTypeName()
		// The `GetTypeName` method on FieldDescriptorProto only
		// returns the path/name of a type if that type is a message or
		// an Enum. For basic types (int, float, etc.) `GetTypeName()`
		// returns an empty string. In that case, we set the newFields
		// type name to be the string representing the type of the
		// field being examined.
		if newField.Type.Name == "" {
			newField.Type.Name = field.Type.String()
		}
		// The label we get back is a number, translate it to a human
		// readable string
		label := int32(field.GetLabel())
		lname := descriptor.FieldDescriptorProto_Label_name[label]
		newField.Label = lname

		newMsg.Fields = append(newMsg.Fields, &newField)
	}
	return &newMsg, nil
}

// NewService creates a new *doctree.ProtoService from a
// descriptor.ServiceDescriptorProto. Additionally requires being passed the
// current *doctree.ProtoFile being defined and a reference to the current
// *doctree.MicroserviceDefinition being defined; this access is necessary so
// that the RequestType and ResponseType fields of each of the methods of this
// service may be set as references to the correct ProtoMessages
func NewService(
	srvc *descriptor.ServiceDescriptorProto,
	curNewFile *doctree.ProtoFile,
	curNewDt *doctree.MicroserviceDefinition) (*doctree.ProtoService, error) {

	newSvc := doctree.ProtoService{}
	newSvc.Name = *srvc.Name

	// Add methods to this service
	for _, meth := range srvc.Method {
		newMeth := doctree.ServiceMethod{}
		newMeth.Name = *meth.Name

		// Set this methods request and responses to point to existing
		// Message types
		reqMsg, err := findMessage(curNewDt, curNewFile, *meth.InputType)
		if reqMsg == nil || err != nil {
			return nil, fmt.Errorf("Couldn't find request message of type '%v' for method '%v'", *meth.InputType, *meth.Name)
		}
		respMsg, err := findMessage(curNewDt, curNewFile, *meth.OutputType)
		if respMsg == nil || err != nil {
			return nil, fmt.Errorf("Couldn't find response message of type '%v' for method '%v'", *meth.InputType, *meth.Name)
		}
		newMeth.RequestType = reqMsg
		newMeth.ResponseType = respMsg

		newSvc.Methods = append(newSvc.Methods, &newMeth)
	}
	return &newSvc, nil
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
