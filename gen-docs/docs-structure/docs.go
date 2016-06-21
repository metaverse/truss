package docs_structure

// Used to give other structs a name and description
type Describable struct {
	Name        string
	Description string
}

type MicroserviceDefinition struct {
	Describable
	Files []*ProtoFile
}

type ProtoFile struct {
	Describable
	Messages []*ProtoMessage
	Enums    []*ProtoEnum
	Services []*ProtoService
}

type ProtoMessage struct {
	Describable
	Fields []*MessageField
}

type MessageField struct {
	Describable
	Type FieldType
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
	RequestType  *ProtoMessage
	ResponseType *ProtoMessage
}
