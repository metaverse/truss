package httptransport

type Method struct {
	Name     string
	Bindings []*Binding
}

type Binding struct {
	Label string
	// PathTemplate is the full path template as it appeared in the http
	// annotation which this binding refers to.
	PathTemplate string
	// BasePath is the longest static portion of the full PathTemplate, and is
	// given the the net/http mux as the path for the route for this binding.
	BasePath string
	Fields   []*Field
}

type Field struct {
	Name string
	// The go-compatible name for this variable, for use in auto generated go
	// code.
	LocalName string
	// The location within the the http request that this field is to be found.
	Location string
	// The protobuf type that this field is of.
	ProtobufType string
	// The type within the Go language that's used to represent the original
	// field that this field refers to.
	GoType string
	// The protobuf label for the original field that this field refers to. Is
	// probably "OPTIONAL", though may be "REPEATED".
	ProtobufLabel string
	// The string form of the function to be used to convert the incoming
	// string msg from a string into it's intended type.
	ConvertFunc string
	// Indicates whether this field represents a basic protobuf type such as
	// one of the ints, floats, strings, bools, etc. Since we can only create
	// automatic marshaling of base types, if this is false a warning is given
	// to the user.
	IsBaseType bool
}
