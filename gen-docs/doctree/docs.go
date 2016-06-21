package doctree

import (
	"fmt"
	descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

var (
	_ = descriptor.MethodDescriptorProto{}
)

func prindent(depth int, format string, args ...interface{}) string {
	s := ""
	for i := 0; i < depth; i++ {
		s += "    "
	}
	return s + fmt.Sprintf(format, args...)
}

// Used to give other structs a name and description, as well as a nice way of
// generating string representations of nested structures
type Describable struct {
	Name        string
	Description string
}

func (x Describable) describe(depth int) string {
	rv := prindent(depth, "Name: %v\n", x.Name)
	rv += prindent(depth, "Desc: %v\n", x.Description)
	return rv
}

type MicroserviceDefinition struct {
	Describable
	Files []*ProtoFile
}

func (x MicroserviceDefinition) describe(depth int) string {
	rv := x.Describable.describe(depth)
	for idx, file := range x.Files {
		rv += prindent(depth, "File %v:\n", idx)
		rv += file.describe(depth + 1)
	}
	return rv
}

// Used to allow for idiomatic 'pkg.New()' functionality
type Doctree MicroserviceDefinition

func (x MicroserviceDefinition) String() string {
	return x.describe(0)
}

type ProtoFile struct {
	Describable
	Messages []*ProtoMessage
	Enums    []*ProtoEnum
	Services []*ProtoService
}

func (x ProtoFile) describe(depth int) string {
	rv := x.Describable.describe(depth)
	for idx, msg := range x.Messages {
		rv += prindent(depth, "Message %v:\n", idx)
		rv += msg.describe(depth + 1)
	}
	for idx, enum := range x.Enums {
		rv += prindent(depth, "Enum %v:\n", idx)
		rv += enum.describe(depth + 1)
	}
	for idx, svc := range x.Services {
		rv += prindent(depth, "Service %v:\n", idx)
		rv += svc.describe(depth + 1)
	}
	return rv
}

type ProtoMessage struct {
	Describable
	Fields []*MessageField
}

func (x ProtoMessage) describe(depth int) string {
	rv := x.Describable.describe(depth)
	for idx, field := range x.Fields {
		rv += prindent(depth, "Field %v:\n", idx)
		rv += field.describe(depth + 1)
	}
	return rv
}

type MessageField struct {
	Describable
	Type FieldType
}

func (x MessageField) describe(depth int) string {
	rv := x.Describable.describe(depth)
	rv += prindent(depth, "Type:\n")
	rv += x.Type.describe(depth + 1)
	return rv
}

type ProtoEnum struct {
	Describable
	Values []*EnumValue
}

type EnumValue struct {
	Describable
	Number int
}

type FieldType struct {
	Describable
	Enum *ProtoEnum
}

type ProtoService struct {
	Describable
	Methods []*ServiceMethod
}

type ServiceMethod struct {
	Describable
	RequestType  string
	ResponseType string
}

func (x ServiceMethod) describe(depth int) string {
	rv := x.Describable.describe(depth)
	rv += prindent(depth, "RequestType: %v\n", x.RequestType)
	rv += prindent(depth, "ResponseType: %v\n", x.ResponseType)
	return rv
}

func New(req *plugin.CodeGeneratorRequest) (MicroserviceDefinition, error) {
	dt := MicroserviceDefinition{}
	for _, file := range req.ProtoFile {
		// Check if this file is one we even should examine
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
		new_file := ProtoFile{}
		new_file.Name = file.GetName()

		for _, msg := range file.MessageType {
			new_msg := ProtoMessage{}
			new_msg.Name = *msg.Name
			for _, field := range msg.Field {
				new_field := MessageField{}
				new_field.Name = *field.Name
				new_field.Type.Name = field.GetTypeName()
				if new_field.Type.Name == "" {
					new_field.Type.Name = field.Type.String()
				}
				new_msg.Fields = append(new_msg.Fields, &new_field)
			}
			new_file.Messages = append(new_file.Messages, &new_msg)
		}

		// Add services to this file
		for _, srvc := range file.Service {
			n_svc := ProtoService{}
			n_svc.Name = *srvc.Name

			// Add methods to this service
			for _, meth := range srvc.Method {
				n_meth := ServiceMethod{}
				n_meth.Name = *meth.Name
				n_meth.RequestType = *meth.InputType
				n_meth.ResponseType = *meth.OutputType
				n_svc.Methods = append(n_svc.Methods, &n_meth)
			}

			new_file.Services = append(new_file.Services, &n_svc)
		}
		dt.Files = append(dt.Files, &new_file)
	}

	return dt, nil
}
