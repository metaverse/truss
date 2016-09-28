package deftree

// build_deftree.go contains the functions for the creation of a deftree and
// it's component structs.

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/pkg/errors"

	"github.com/TuneLab/go-truss/deftree/svcparse"
	"github.com/TuneLab/go-truss/truss/protostage"
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

// New accepts a Protobuf plugin.CodeGeneratorRequest and the contents of the
// file containing the service declaration and returns a Deftree struct
func New(req *plugin.CodeGeneratorRequest, serviceFile io.Reader) (Deftree, error) {
	dt := MicroserviceDefinition{}
	dt.SetName(findDeftreePackage(req))

	var svc *ProtoService
	var serviceFileName string
	for _, file := range req.ProtoFile {
		// Check if this file is one we even should examine, and if it's not,
		// skip it
		if file.GetPackage() != findDeftreePackage(req) {
			continue
		}
		// This is a file we are meant to examine, so contine with its creation
		// in the Deftree
		newFile, err := NewFile(file, &dt)
		if err != nil {
			return nil, errors.Wrapf(err, "file creation of %q failed", file.GetName())
		}

		if len(newFile.Services) > 0 {
			svc = newFile.Services[0]
			serviceFileName = newFile.GetName()
		}

		dt.Files = append(dt.Files, newFile)
	}

	// AssociateComments goes through the comments in the passed in protobuf
	// CodeGeneratorRequest, figures out which node within the mostly-assembled
	// deftree each comment corresponds with, then uses the `SetDescription`
	// method of each node to set it's description to the comment.
	// The implementation of this function is in deftree/associate_comments.go
	AssociateComments(&dt, req)

	err := addHttpOptions(&dt, svc, serviceFile)
	if err != nil {
		log.WithError(err).Warnf("Error found while parsing file %v", serviceFileName)
		log.Warnf("Due to the above warning(s), http options and bindings where not parsed and will not be present in the generated documentation.")
	}

	return &dt, nil
}

func NewFromString(def string) (Deftree, error) {
	const defFileName = "definition.proto"

	protoDir, err := ioutil.TempDir("", "truss-deftree-")
	if err != nil {
		return nil, errors.Wrap(err, "could not create temp directory to store proto definition")
	}
	defer os.RemoveAll(protoDir)

	err = ioutil.WriteFile(filepath.Join(protoDir, defFileName), []byte(def), 0666)
	if err != nil {
		return nil, errors.Wrap(err, "could not write proto definition to file")
	}

	err = protostage.Stage(protoDir)
	if err != nil {
		return nil, errors.Wrap(err, "unable to prepare filesystem for generating deftree")
	}

	req, svcFile, err := protostage.Compose([]string{defFileName}, protoDir)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get CodeGeneratorRequest for generating deftree")
	}

	deftree, err := New(req, svcFile)
	if err != nil {
		return nil, errors.Wrap(err, "can not create new deftree")
	}

	return deftree, nil
}

// Finds the package name of the proto files named on the command line
func findDeftreePackage(req *plugin.CodeGeneratorRequest) string {
	for _, cmdFile := range req.GetFileToGenerate() {
		for _, protoFile := range req.GetProtoFile() {
			if protoFile.GetName() == cmdFile {
				return protoFile.GetPackage()
			}
		}
	}
	return ""
}

// Build a new deftree.File struct
func NewFile(
	pfile *descriptor.FileDescriptorProto,
	curNewDt *MicroserviceDefinition) (*ProtoFile, error) {

	newFile := ProtoFile{}
	newFile.Name = pfile.GetName()

	for _, enum := range pfile.EnumType {
		newEnum, err := NewEnum(enum)
		if err != nil {
			return nil, errors.Wrapf(err, "error converting enum %q", enum.GetName())
		}
		newFile.Enums = append(newFile.Enums, newEnum)
	}

	for _, msg := range pfile.MessageType {
		newMsg, err := NewMessage(msg)
		if err != nil {
			return nil, errors.Wrapf(err, "error converting message %q", msg.GetName())
		}
		newFile.Messages = append(newFile.Messages, newMsg)
	}

	for _, srvc := range pfile.Service {
		newSvc, err := NewService(srvc, &newFile, curNewDt)
		if err != nil {
			return nil, errors.Wrapf(err, "error converting service %q", srvc.GetName())
		}
		// Set the new services FullyQualifiedName here so that we don't have
		// to pass around additional references to pfile.
		newSvc.FullyQualifiedName = "." + pfile.GetPackage() + "." + newSvc.Name
		newFile.Services = append(newFile.Services, newSvc)
	}

	return &newFile, nil
}

// NewEnum returns a *ProtoEnum created from a
// *descriptor.EnumDescriptorProto
func NewEnum(enum *descriptor.EnumDescriptorProto) (*ProtoEnum, error) {
	newEnum := ProtoEnum{}

	newEnum.SetName(enum.GetName())
	// Add values to this enum
	for _, val := range enum.GetValue() {
		nval := EnumValue{}
		nval.SetName(val.GetName())
		nval.Number = int(val.GetNumber())
		newEnum.Values = append(newEnum.Values, &nval)
	}

	return &newEnum, nil
}

// NewMessage returns a *ProtoMessage created from a
// *descriptor.DescriptorProto
func NewMessage(msg *descriptor.DescriptorProto) (*ProtoMessage, error) {
	newMsg := ProtoMessage{}
	newMsg.Name = *msg.Name
	// Add fields to this message
	for _, field := range msg.Field {
		newField := MessageField{}
		newField.Number = int(field.GetNumber())
		newField.Name = *field.Name
		newField.Type.Name = getCorrectTypeName(field)
		// The label we get back is a number, translate it to a human
		// readable string
		label := int32(field.GetLabel())
		lname := descriptor.FieldDescriptorProto_Label_name[label]
		newField.Label = lname

		newMsg.Fields = append(newMsg.Fields, &newField)
	}
	return &newMsg, nil
}

// Finds a message given a fully qualified name to that message. The provided
// path may be either a fully qualfied name of a message, or just the bare name
// for a message.
func findMessage(md *MicroserviceDefinition, newFile *ProtoFile, path string) (*ProtoMessage, error) {
	if path[0] == '.' {
		parts := strings.Split(path, ".")
		for _, file := range md.Files {
			for _, msg := range file.Messages {
				if parts[2] == msg.GetName() {
					return msg, nil
				}
			}
		}
		for _, msg := range newFile.Messages {
			if parts[2] == msg.GetName() {
				return msg, nil
			}
		}
	} else {
		for _, msg := range newFile.Messages {
			if path == msg.GetName() {
				return msg, nil
			}
		}
	}
	return nil, fmt.Errorf("couldn't find message")
}

// NewService creates a new *ProtoService from a
// descriptor.ServiceDescriptorProto. Additionally requires being passed the
// current *ProtoFile being defined and a reference to the current
// *MicroserviceDefinition being defined; this access is necessary so that the
// RequestType and ResponseType fields of each of the methods of this service
// may be set as references to the correct ProtoMessages
func NewService(
	srvc *descriptor.ServiceDescriptorProto,
	curNewFile *ProtoFile,
	curNewDt *MicroserviceDefinition) (*ProtoService, error) {

	newSvc := ProtoService{}
	newSvc.Name = *srvc.Name

	// Add methods to this service
	for _, meth := range srvc.Method {
		newMeth := ServiceMethod{}
		newMeth.Name = *meth.Name

		// Set this methods request and responses to point to existing
		// Message types
		reqMsg, err := findMessage(curNewDt, curNewFile, *meth.InputType)
		if reqMsg == nil || err != nil {
			return nil, fmt.Errorf("couldn't find request message of type '%v' for method '%v'", *meth.InputType, *meth.Name)
		}
		respMsg, err := findMessage(curNewDt, curNewFile, *meth.OutputType)
		if respMsg == nil || err != nil {
			return nil, fmt.Errorf("couldn't find response message of type '%v' for method '%v'", *meth.InputType, *meth.Name)
		}
		newMeth.RequestType = reqMsg
		newMeth.ResponseType = respMsg

		newSvc.Methods = append(newSvc.Methods, &newMeth)
	}
	return &newSvc, nil
}

// getCorrectTypeName returns the correct name for the type of the given
// FieldDescriptorProto. The GetTypeName method on FieldDescriptorProto only
// returns the path/name of a type if that type is a message or an Enum. For
// basic types (int, float, etc.) GetTypeName() returns an empty string. In
// that case, we set the newFields type name to be the string representing the
// type of the field being examined.
func getCorrectTypeName(p *descriptor.FieldDescriptorProto) string {
	rv := p.GetTypeName()

	if rv == "" {
		rv = p.Type.String()
	}
	return rv
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

// convertSvcparse converts the structures returned by the service parser into
// the equivalent representation as deftree structures. At this time,
// convertSvcparse won't ever return an error, but that may change at any time,
// so please do not ignore the error on this function!
func convertSvcparse(parsedSvc *svcparse.Service) (*ProtoService, error) {
	rv := &ProtoService{}
	rv.SetName(parsedSvc.Name)

	for _, pm := range parsedSvc.Methods {
		m := &ServiceMethod{
			Name:        pm.Name,
			Description: scrubComments(pm.Description),
		}

		m.RequestType = &ProtoMessage{
			Name: pm.RequestType,
		}
		m.ResponseType = &ProtoMessage{
			Name: pm.ResponseType,
		}

		for _, pb := range pm.HTTPBindings {
			b := &MethodHttpBinding{
				Description: scrubComments(pb.Description),
			}
			for _, pf := range pb.Fields {
				f := &BindingField{
					Name:        pf.Name,
					Description: scrubComments(pf.Description),
					Kind:        pf.Kind,
					Value:       pf.Value,
				}
				b.Fields = append(b.Fields, f)
			}
			m.HttpBindings = append(m.HttpBindings, b)
		}
		rv.Methods = append(rv.Methods, m)
	}

	return rv, nil
}

// Parse the protobuf files for comments surrounding http options, then add
// those to the Deftree in place.
func addHttpOptions(dt Deftree, svc *ProtoService, protoFile io.Reader) error {

	lex := svcparse.NewSvcLexer(protoFile)
	ps, err := svcparse.ParseService(lex)
	if err != nil {
		return errors.Wrapf(err, "error while parsing http options for the %v service definition", svc.GetName())
	}
	parsedSvc, err := convertSvcparse(ps)
	if err != nil {
		return errors.Wrapf(err, "error while converting result of service parser for the %v service definition", svc.GetName())
	}

	for _, pmeth := range parsedSvc.Methods {
		meth := svc.GetByName(pmeth.GetName()).(*ServiceMethod)
		meth.HttpBindings = pmeth.HttpBindings
	}

	// Assemble the http parameters for each http binding
	err = Assemble(dt)
	if err != nil {
		return errors.Wrap(err, "could not assemble http parameters for each http binding")
	}

	return nil
}

// Searches through the files in the request and returns the path to the first
// one which contains a service declaration. If no file in the request contains
// a service, returns an empty string.
func FindServiceFile(req *plugin.CodeGeneratorRequest) string {
	svcFiles := []string{}
	// Since the names of proto files in FileDescriptorProto's don't contain
	// the path, we have to find the first one with a service, then find its
	// actual relative path by searching the slice `FileToGenerate`.
	for _, file := range req.GetProtoFile() {
		if len(file.GetService()) > 0 {
			svcFiles = append(svcFiles, file.GetName())
		}
	}
	for _, file := range req.GetFileToGenerate() {
		for _, svcF := range svcFiles {
			if strings.Contains(file, svcF) {
				return file
			}
		}
	}
	return ""
}
