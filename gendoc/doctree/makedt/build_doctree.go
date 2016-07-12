package makedt

import (
	"fmt"
	"os"
	"strings"

	"github.com/TuneLab/gob/gendoc/doctree"
	"github.com/TuneLab/gob/gendoc/doctree/httpopts"
	"github.com/TuneLab/gob/gendoc/svcparse"

	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

func lastName(in string) string {
	sl := strings.Split(in, ".")
	return sl[len(sl)-1]
}

// New accepts a Protobuf CodeGeneratorRequest and returns a Doctree struct
func New(req *plugin.CodeGeneratorRequest) (doctree.Doctree, error) {
	dt := doctree.MicroserviceDefinition{}
	for _, file := range req.ProtoFile {
		// Check if this file is one we even should examine, and if it's not,
		// skip it
		fname := file.GetName()
		should_examine := false
		for _, goodf := range req.FileToGenerate {
			if fname == goodf {
				should_examine = true
			}
		}
		if should_examine == false {
			continue
		}

		// This is a file we are meant to examine, so contine with it's
		// creation in the Doctree
		new_file := doctree.ProtoFile{}
		new_file.Name = file.GetName()

		if dt.Name == "" {
			dt.Name = *file.Package
		} else {
			if dt.Name != *file.Package {
				panic("Package name of specified protobuf definitions differ.")
			}
		}

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
				new_msg.Fields = append(new_msg.Fields, &new_field)
			}
			new_file.Messages = append(new_file.Messages, &new_msg)
		}

		// Add services to this file
		for _, srvc := range file.Service {
			n_svc := doctree.ProtoService{}
			n_svc.Name = *srvc.Name

			// Add methods to this service
			for _, meth := range srvc.Method {
				n_meth := doctree.ServiceMethod{}
				n_meth.Name = *meth.Name

				// Set this methods request and responses to point to existing
				// Message types
				req_msg := new_file.GetByName(lastName(*meth.InputType))
				if req_msg == nil {
					panic(fmt.Sprintf("Couldn't find message type for '%v'\n", *meth.InputType))
				}
				resp_msg := new_file.GetByName(lastName(*meth.OutputType))
				if req_msg == nil {
					panic(fmt.Sprintf("Couldn't find message type for '%v'\n", *meth.OutputType))
				}
				n_meth.RequestType = req_msg.(*doctree.ProtoMessage)
				n_meth.ResponseType = resp_msg.(*doctree.ProtoMessage)

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

// Parse the protobuf files for comments surrounding http options, then add
// those to the Doctree in place.
func addHttpOptions(dt doctree.Doctree, req *plugin.CodeGeneratorRequest) {
	for _, fname := range req.FileToGenerate {
		f, _ := os.Open(fname)
		lex := svcparse.NewSvcLexer(f)
		parsed_svc, err := svcparse.ParseService(lex)

		if err != nil {
			panic(err)
		}

		svc := dt.GetByName(fname).GetByName(parsed_svc.GetName()).(*doctree.ProtoService)
		for _, pmeth := range parsed_svc.Methods {
			meth := svc.GetByName(pmeth.GetName()).(*doctree.ServiceMethod)
			meth.HttpBindings = pmeth.HttpBindings
		}
	}
	// Assemble the http parameters for each http binding
	httpopts.Assemble(dt)
}
