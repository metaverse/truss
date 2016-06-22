// Doctree, which stands for "documentation tree", creates a tree of nodes
// representing the components of a serviced defined through Protobuf
// definition files. The tree is composed of nodes fulfilling the `Describable`
// interface, with the root node fulfilling the `Doctree` interface. The
// `Doctree` interface is a superset of the `Describable` interface.
//
// The main entrypoint for the Doctree package is the `New` function, which
// takes a Protobuf `CodeGeneratorRequest` struct and creates a Doctree
// representing all the documentation from the `CodeGeneratorRequest`.
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

// prindent is a utility function for creating a formatted string with a given
// amount of indentation.
func prindent(depth int, format string, args ...interface{}) string {
	s := ""
	for i := 0; i < depth; i++ {
		s += "    "
	}
	return s + fmt.Sprintf(format, args...)
}

// Describable offers an interface for traversing a Doctree and finding
// information from the nodes within it.
type Describable interface {
	// The "Name" of this describable
	GetName() string
	SetName(string)
	// GetDescription returns the description of this describable
	GetDescription() string
	SetDescription(string)
	// describe causes a Describable to generate a string representing itself.
	// The integer argument is used as the 'depth' that this Describable sits
	// within a tree of Describable structs, allowing it to print it's
	// information with proper indentation. If called recursively, allows for
	// printing of a structured tree-style view of a tree of Describables.
	describe(int) string
	// GetByName allows one to query a Describable to see if it has a child
	// Describable in any of it's collections.
	GetByName(string) Describable
}

// Doctree is the root interface for this package, and is chiefly implemented
// by MicroserviceDefinition. See MicroserviceDefinition for further
// documentation on these Methods.
type Doctree interface {
	Describable
	SetComment([]string, string)
	String() string
}

// describable is a  concrete implementation of the `Describable` interface, to
// allow for nice convenient inheritance with concrete default methods.
type describable struct {
	Name        string
	Description string
}

func (self *describable) GetName() string {
	return self.Name
}

func (self *describable) SetName(s string) {
	self.Name = s
}

func (self *describable) describe(depth int) string {
	rv := prindent(depth, "Name: %v\n", self.Name)
	rv += prindent(depth, "Desc: %v\n", self.Description)
	return rv
}

func (self *describable) GetDescription() string {
	return self.Description
}

func (self *describable) SetDescription(d string) {
	self.Description = d
}

func (self *describable) GetByName(s string) Describable {
	return nil
}

// MicroserviceDefinition is the root node for any particular `Doctree`
type MicroserviceDefinition struct {
	describable
	Files []*ProtoFile
}

func (self *MicroserviceDefinition) describe(depth int) string {
	rv := self.describable.describe(depth)
	for idx, file := range self.Files {
		rv += prindent(depth, "File %v:\n", idx)
		rv += file.describe(depth + 1)
	}
	return rv
}

// GetByName returns any ProtoFile structs it my have with a matching `Name`.
func (self *MicroserviceDefinition) GetByName(name string) Describable {
	for _, file := range self.Files {
		if file.Name == name {
			return file
		}
	}
	return nil
}

// SetComment changes the node at the given 'name-path' to have a description
// of `comment_body`. `name-path` is a series of names of describable objects
// each found within the previous, accessed by recursively calling `GetByName`
// on the result of the last call, beginning with this MicroserviceDefinition.
// Once the final Describable object is found, the `description` field of that
// struct is set to `comment_body`.
func (self *MicroserviceDefinition) SetComment(namepath []string, comment_body string) {
	fmt.Fprintf(os.Stderr, "%v\n", comment_body)
	var cur_node Describable
	cur_node = self
	for _, name := range namepath {
		new_node := cur_node.GetByName(name)
		if new_node == nil {
			panic(fmt.Sprintf("New node is nil, namepath: '%v' cur_node: '%v'\n", namepath, cur_node))
		}
		cur_node = new_node
	}
	cur_node.SetDescription(comment_body)
}

// String kicks off the recursive call to `describe` within the tree of
// Describables, returning a string showing the structured view of the tree.
func (self *MicroserviceDefinition) String() string {
	return self.describe(0)
}

type ProtoFile struct {
	describable
	Messages []*ProtoMessage
	Enums    []*ProtoEnum
	Services []*ProtoService
}

func (self *ProtoFile) describe(depth int) string {
	rv := self.describable.describe(depth)
	for idx, msg := range self.Messages {
		rv += prindent(depth, "Message %v:\n", idx)
		rv += msg.describe(depth + 1)
	}
	for idx, enum := range self.Enums {
		rv += prindent(depth, "Enum %v:\n", idx)
		rv += enum.describe(depth + 1)
	}
	for idx, svc := range self.Services {
		rv += prindent(depth, "Service %v:\n", idx)
		rv += svc.describe(depth + 1)
	}
	return rv
}

func (self *ProtoFile) GetByName(name string) Describable {
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

func (self *ProtoMessage) describe(depth int) string {
	rv := self.describable.describe(depth)
	for idx, field := range self.Fields {
		rv += prindent(depth, "Field %v:\n", idx)
		rv += field.describe(depth + 1)
	}
	return rv
}

func (self *ProtoMessage) GetByName(name string) Describable {
	for _, field := range self.Fields {
		if field.GetName() == name {
			return field
		}
	}
	return nil
}

type MessageField struct {
	describable
	Type   FieldType
	Number int
}

func (self *MessageField) describe(depth int) string {
	rv := self.describable.describe(depth)
	rv += prindent(depth, "Number: %v\n", self.Number)
	rv += prindent(depth, "Type:\n")
	rv += self.Type.describe(depth + 1)
	return rv
}

type ProtoEnum struct {
	describable
	Values []*EnumValue
}

func (self *ProtoEnum) describe(depth int) string {
	rv := self.describable.describe(depth)
	for idx, val := range self.Values {
		rv += prindent(depth, "Value %v:\n", idx)
		rv += val.describe(depth + 1)
	}
	return rv
}

type EnumValue struct {
	describable
	Number int
}

func (self *EnumValue) describe(depth int) string {
	rv := self.describable.describe(depth)
	rv += prindent(depth, "Number: %v\n", self.Number)
	return rv
}

type FieldType struct {
	describable
	Enum *ProtoEnum
}

type ProtoService struct {
	describable
	Methods []*ServiceMethod
}

func (self *ProtoService) describe(depth int) string {
	rv := self.describable.describe(depth)
	for idx, meth := range self.Methods {
		rv += prindent(depth, "Method %v:\n", idx)
		rv += meth.describe(depth + 1)
	}
	return rv
}

func (self *ProtoService) GetByName(name string) Describable {
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

func (self *ServiceMethod) describe(depth int) string {
	rv := self.describable.describe(depth)
	rv += prindent(depth, "RequestType: %v\n", self.RequestType.GetName())
	rv += prindent(depth, "ResponseType: %v\n", self.ResponseType.GetName())
	return rv
}

func (self *ServiceMethod) GetByName(name string) Describable {
	if name == self.RequestType.GetName() {
		return &self.RequestType
	}
	if name == self.ResponseType.GetName() {
		return &self.ResponseType
	}
	return nil
}

// New accepts a Protobuf CodeGeneratorRequest and returns a Doctree struct
func New(req *plugin.CodeGeneratorRequest) (Doctree, error) {
	dt := MicroserviceDefinition{}
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
		new_file := ProtoFile{}
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
			new_enum := ProtoEnum{}
			new_enum.SetName(enum.GetName())
			for _, val := range enum.GetValue() {
				// Add values to this enum
				n_val := EnumValue{}
				n_val.SetName(val.GetName())
				n_val.Number = int(val.GetNumber())
				new_enum.Values = append(new_enum.Values, &n_val)
			}
			new_file.Enums = append(new_file.Enums, &new_enum)
		}

		// Add messages to this file
		for _, msg := range file.MessageType {
			new_msg := ProtoMessage{}
			new_msg.Name = *msg.Name
			// Add fields to this message
			for _, field := range msg.Field {
				new_field := MessageField{}
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

	return &dt, nil
}
