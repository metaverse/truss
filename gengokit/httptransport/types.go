package httptransport

// Method contains the distillation of information within an
// svcdef.ServiceMethod that's useful for templating http transport.
type Method struct {
	Name string
	// RequestType is the name of type of the Request, e.g. *EchoRequest
	RequestType  string
	ResponseType string
	Bindings     []*Binding
}

// Binding contains the distillation of information within an
// svcdef.HTTPBinding that's useful for templating http transport.
type Binding struct {
	// Label is the name of this method, plus the english word for the index of
	// this binding in this methods slice of bindings. So if this binding where
	// the first binding in the slice of bindings for the method "Sum", the
	// label for this binding would be "SumZero". If it where the third
	// binding, it would be named "SumTwo".
	Label string
	// PathTemplate is the full path template as it appeared in the http
	// annotation which this binding refers to.
	PathTemplate string
	// BasePath is the longest static portion of the full PathTemplate, and is
	// given to the net/http mux as the path for the route for this binding.
	BasePath string
	Verb     string
	Fields   []*Field
	// A pointer back to the parent method of this binding. Used within some
	// binding methods
	Parent *Method
}

// Field contains the distillation of information within an svcdef.Field that's
// useful for templating http transport.
type Field struct {
	Name           string
	QueryParamName string
	// The name of this field, but passed through the CamelCase function.
	// Removes underscores, adds camelcase; "client_id" becomes "ClientId".
	CamelName string
	// The name of this field, but run through the camelcase function and with
	// the first letter lowercased. "example_name" becomes "exampleName".
	// LowCamelName is how the names of fields should appear when marshaled to
	// JSON, according to the gRPC language guide.
	LowCamelName string
	// The go-compatible name for this variable, for use in auto generated go
	// code.
	LocalName string
	// The location within the the http request that this field is to be found.
	Location string
	// The type within the Go language that's used to represent the original
	// field that this field refers to.
	GoType string
	// The string form of the function to be used to convert the incoming
	// string msg from a string into it's intended type.
	ConvertFunc string
	// Used in determining if a convert func will need error checking logic
	ConvertFuncNeedsErrorCheck bool
	// The string form of a type cast from 64 to 32bit if the GoType is 32bit
	// as the ConvertFunc will always use return a 64bit type
	TypeConversion string
	// Indicates whether this field represents a basic protobuf type such as
	// one of the ints, floats, strings, bools, etc. Since we can only create
	// automatic marshaling of base types, if this is false a warning is given
	// to the user.
	IsBaseType bool
	// Protobuf Enums need to be handled uniquely when parsing queryparameters
	IsEnum bool
	// Repeated is true if this arg corresponds to a protobuf field which is
	// given an identifier of "repeated", meaning it will represented in Go as
	// a slice of it's type.
	Repeated bool
}
