package doctree

import (
	"fmt"
	"os"

	descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

var (
	_ = descriptor.MethodDescriptorProto{}
	_ = os.Stderr
)

func prindent(depth int, format string, args ...interface{}) string {
	s := ""
	for i := 0; i < depth; i++ {
		s += "    "
	}
	return s + fmt.Sprintf(format, args...)
}

type Describable interface {
	GetName() string
	SetName(string)
	GetDescription() string
	SetDescription(string)
	describe(int) string
	GetByName(string) Describable
}

// The concrete implementation of the Interface, to allow for nice convenient
// inheritance
type describable struct {
	Name        string
	Description string
}

func (self describable) GetName() string {
	return self.Name
}

func (self describable) SetName(s string) {
	self.Name = s
}

func (self describable) describe(depth int) string {
	rv := prindent(depth, "Name: %v\n", self.Name)
	rv += prindent(depth, "Desc: %v\n", self.Description)
	return rv
}

func (self describable) GetDescription() string {
	return self.Description
}

func (self *describable) SetDescription(d string) {
	self.Description = d
}

func (self describable) GetByName(s string) Describable {
	return nil
}

func NewDescribable() Describable {
	return &describable{}
}

type MicroserviceDefinition struct {
	describable
	Files []*ProtoFile
}

func (x MicroserviceDefinition) describe(depth int) string {
	rv := x.describable.describe(depth)
	for idx, file := range x.Files {
		rv += prindent(depth, "File %v:\n", idx)
		rv += file.describe(depth + 1)
	}
	return rv
}

func (x MicroserviceDefinition) GetByName(name string) Describable {
	for _, file := range x.Files {
		if file.Name == name {
			return file
		}
	}
	return nil
}

// Set the node at the given 'name-path' to have a description of `comment_body`
func (self *MicroserviceDefinition) SetComment(namepath []string, comment_body string) {
	fmt.Fprintf(os.Stderr, "%v\n", comment_body)
	var cur_node Describable
	cur_node = self
	for _, name := range namepath {
		new_node := cur_node.GetByName(name)
		if new_node == nil {
			//panic("The new node is nil, this is bad!")
			panic(fmt.Sprintf("New node is nil, namepath: '%v' cur_node: '%v'\n", namepath, cur_node))
		}
		//fmt.Fprintf(os.Stderr, "Name: '%v', Cur_node: '%v', new_node: '%v'\n", name, cur_node.GetName(), new_node.GetName())
		cur_node = new_node
	}
	cur_node.SetDescription(comment_body)
	//fmt.Fprintf(os.Stderr, "%v\n", self.String())
}

func (x MicroserviceDefinition) String() string {
	return x.describe(0)
}

type ProtoFile struct {
	//Describable
	describable
	Messages []*ProtoMessage
	Enums    []*ProtoEnum
	Services []*ProtoService
}

func (x ProtoFile) describe(depth int) string {
	rv := x.describable.describe(depth)
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

func (self ProtoFile) GetByName(name string) Describable {
	for _, msg := range self.Messages {
		if msg.GetName() == name {
			return msg
		}
	}
	for _, enum := range self.Enums {
		if enum.GetName() == name {
			return enum
		}
	}
	for _, svc := range self.Services {
		if svc.GetName() == name {
			return svc
		}
	}
	return nil
}

type ProtoMessage struct {
	describable
	Fields []*MessageField
}

func (x ProtoMessage) describe(depth int) string {
	rv := x.describable.describe(depth)
	for idx, field := range x.Fields {
		rv += prindent(depth, "Field %v:\n", idx)
		rv += field.describe(depth + 1)
	}
	return rv
}

func (self ProtoMessage) GetByName(name string) Describable {
	for _, field := range self.Fields {
		if field.GetName() == name {
			return field
		}
	}
	return nil
}

type MessageField struct {
	describable
	Type FieldType
}

func (x MessageField) describe(depth int) string {
	rv := x.describable.describe(depth)
	rv += prindent(depth, "Type:\n")
	rv += x.Type.describe(depth + 1)
	return rv
}

type ProtoEnum struct {
	describable
	Values []*EnumValue
}

type EnumValue struct {
	describable
	Number int
}

type FieldType struct {
	describable
	Enum *ProtoEnum
}

type ProtoService struct {
	describable
	Methods []*ServiceMethod
}

func (self ProtoService) describe(depth int) string {
	rv := self.describable.describe(depth)
	for idx, meth := range self.Methods {
		rv += prindent(depth, "Method %v:\n", idx)
		rv += meth.describe(depth + 1)
	}
	return rv
}

func (self ProtoService) GetByName(name string) Describable {
	for _, meth := range self.Methods {
		if meth.GetName() == name {
			return meth
		}
	}
	return nil
}

type ServiceMethod struct {
	describable
	RequestType  ProtoMessage
	ResponseType ProtoMessage
}

func (x ServiceMethod) describe(depth int) string {
	rv := x.describable.describe(depth)
	rv += prindent(depth, "RequestType: %v\n", x.RequestType.GetName())
	rv += prindent(depth, "ResponseType: %v\n", x.ResponseType.GetName())
	return rv
}

func (self ServiceMethod) GetByName(name string) Describable {
	if name == self.RequestType.GetName() {
		return &self.RequestType
	}
	if name == self.ResponseType.GetName() {
		return &self.ResponseType
	}
	return nil
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

		if dt.Name == "" {
			dt.Name = *file.Package
		} else {
			if dt.Name != *file.Package {
				panic("Package name of specified protobuf definitions differ.")
			}
		}

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
				n_meth.RequestType.SetName(*meth.InputType)
				n_meth.ResponseType.SetName(*meth.OutputType)
				n_svc.Methods = append(n_svc.Methods, &n_meth)
			}

			new_file.Services = append(new_file.Services, &n_svc)
		}
		dt.Files = append(dt.Files, &new_file)
	}

	return dt, nil
}
