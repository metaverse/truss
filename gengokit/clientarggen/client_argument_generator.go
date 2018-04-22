// Package clientarggen collects information for templating the code in a
// truss-generated client which marshals command line flags into message fields
// for each service. Functions and fields in clientargen are called by
// templates in protoc-gen-truss-gokit/template/
package clientarggen

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"unicode"

	gogen "github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/pkg/errors"

	"github.com/tuneinc/truss/svcdef"
)

// A collection of the necessary information for generating basic business
// logic in the client. This business logic will allow the generated client to:
//     1. Have command line flags of the correct go types
//     2. Place correctly named and typed arguments in each request method
//     3. Create a request struct with the the function arguments as fields
//
// Since we can only automatically generate handlers for some basic types, if a
// ClientArg is for a field that's not a base type (such as if it's an embedded
// message) then the developer is going to have to write a handler for that
// themselves.
type ClientArg struct {
	// Name contains the name of the arg as it appeared in the original
	// protobuf definition.
	Name string

	// FlagName is the name of the command line flag to be passed to set this
	// argument.
	FlagName string
	// FlagArg is the name of the Go variable that will hold the result of
	// parsing the command line flag.
	FlagArg string
	// FlagType is the the type provided to the flag library and determines the
	// Go type of the variable named FlagArg.
	FlagType string
	// FlagConvertFunc is the code for invoking the flag library to parse the
	// command line parameter named FlagName and store it in the Go variable
	// FlagArg.
	FlagConvertFunc string

	// GoArg is the Go variable that is of the same type as the corresponding
	// field of the message struct.
	GoArg string
	// GoType is the type of this arg's field on the message struct.
	GoType string
	// GoConvertInvoc is the code for initializing GoArg with either a typecast
	// from FlagArg or the invocation of a json unmarshal function.
	GoConvertInvoc string

	// IsBaseType is true if this arg corresponds to a protobuf field which is
	// any of the basic types, or a basic type but repeated. If this the field
	// was a nested message or a map, IsBaseType is false.
	IsBaseType bool
	// Repeated is true if this arg corresponds to a protobuf field which is
	// given an identifier of "repeated", meaning it will represented in Go as
	// a slice of it's type.
	Repeated bool
	// Enum is true if this arg corresponds to a protobuf field which is an
	// enum type.
	Enum bool
}

// MethodArgs is a struct containing a slice of all the ClientArgs for this
// Method.
type MethodArgs struct {
	Args []*ClientArg
}

// FunctionArgs returns a string for the arguments of the function signature
// for a given method. E.g. If there where a method "Sum" with arguments "A"
// and "B", this would generate the arguments portion of a function declaration
// like this:
//
//     func Sum(ASum int64, BSum int64) (pb.SumRequest, error) {
//              └────────────────────┘
func (m *MethodArgs) FunctionArgs() string {
	tmp := []string{}
	for _, a := range m.Args {
		tmp = append(tmp, fmt.Sprintf("%s %s", a.GoArg, a.GoType))
	}
	return strings.Join(tmp, ", ")
}

// CallArgs returns a string for the variables to pass to a function
// implementing this method. Example:
//
//     request, _ := clientHandler.Sum(ASum,  BSum)
//                                     └──────────┘
func (m *MethodArgs) CallArgs() string {
	tmp := []string{}
	for _, a := range m.Args {
		tmp = append(tmp, a.GoArg)
	}
	return strings.Join(tmp, ", ")
}

// MarshalFlags returns the code for intantiating the GoArgs for this method
// while calling the code to marshal flag args into the correct go types.
// Example:
//
//     ASum := int32(flagASum)
//     └─────────────────────┘
func (m *MethodArgs) MarshalFlags() string {
	tmp := []string{}
	for _, a := range m.Args {
		tmp = append(tmp, a.GoConvertInvoc)
	}
	return strings.Join(tmp, "\n")
}

// ClientServiceArgs is a map from the name of a method to a slice of all the
// ClientArgs for that method.
type ClientServiceArgs struct {
	MethArgs map[string]*MethodArgs
}

// AllFlags returns a string that is all the flag declarations for all
// arguments of all methods, separated by newlines. This is used in the
// template to declare all the flag arguments for a client at once, and without
// doing all this iteration in a template where it would be much less
// understandable.
func (c *ClientServiceArgs) AllFlags() string {
	tmp := []string{}
	for _, m := range c.MethArgs {
		for _, a := range m.Args {
			tmp = append(tmp, a.FlagConvertFunc)
		}
	}
	return strings.Join(tmp, "\n")
}

var ProtoToGoTypeMap = map[string]string{
	"TYPE_DOUBLE":   "float64",
	"TYPE_FLOAT":    "float32",
	"TYPE_INT64":    "int64",
	"TYPE_UINT64":   "uint64",
	"TYPE_INT32":    "int32",
	"TYPE_UINT32":   "uint32",
	"TYPE_FIXED64":  "uint64",
	"TYPE_FIXED32":  "uint32",
	"TYPE_BOOL":     "bool",
	"TYPE_STRING":   "string",
	"TYPE_SFIXED32": "int32",
	"TYPE_SFIXED64": "int64",
	"TYPE_SINT32":   "int32",
	"TYPE_SINT64":   "int64",
}

// New creates a ClientServiceArgs struct containing all the arguments for all
// the methods of a given RPC.
func New(svc *svcdef.Service) *ClientServiceArgs {
	svcArgs := ClientServiceArgs{
		MethArgs: make(map[string]*MethodArgs),
	}
	for _, meth := range svc.Methods {
		m := MethodArgs{}
		// TODO implement correct map support in client argument generation
		for _, field := range meth.RequestType.Message.Fields {
			if field.Type.Map != nil {
				continue
			}
			newArg := newClientArg(meth.Name, field)
			m.Args = append(m.Args, newArg)
		}
		svcArgs.MethArgs[meth.Name] = &m
	}

	return &svcArgs
}

// newClientArg returns a ClientArg generated from the provided method name and MessageField
func newClientArg(methName string, field *svcdef.Field) *ClientArg {
	newArg := ClientArg{}
	newArg.Name = lowCamelName(field.Name)

	if field.Type.ArrayType {
		newArg.Repeated = true
	}

	newArg.FlagName = fmt.Sprintf("%s", strings.ToLower(field.Name))
	newArg.FlagArg = fmt.Sprintf("flag%s%s", gogen.CamelCase(field.Name), gogen.CamelCase(methName))

	if field.Type.Enum != nil {
		newArg.Enum = true
	}
	// Determine the FlagType and flag invocation
	var ft string
	if field.Type.Message == nil && field.Type.Enum == nil && field.Type.Map == nil {
		ft = field.Type.Name
		newArg.IsBaseType = true
	} else {
		// For types outside the base types, have flag treat them as strings
		ft = "string"
		newArg.IsBaseType = false
	}
	if newArg.Repeated {
		ft = "string"
	}
	newArg.FlagType = ft
	newArg.FlagConvertFunc = createFlagConvertFunc(newArg, methName)

	newArg.GoArg = fmt.Sprintf("%s%s", gogen.CamelCase(newArg.Name), gogen.CamelCase(methName))
	// For types outside the base types, treat them as strings
	if newArg.IsBaseType {
		//newArg.GoType = ProtoToGoTypeMap[field.Type.GetName()]
		newArg.GoType = field.Type.Name
	} else {
		newArg.GoType = "pb." + field.Type.Name
	}
	// The GoType is a slice of the GoType if it's a repeated field
	if newArg.Repeated {
		if newArg.IsBaseType || newArg.Enum {
			newArg.GoType = "[]" + newArg.GoType
		} else {
			newArg.GoType = "[]*" + newArg.GoType
		}
	}

	newArg.GoConvertInvoc = goConvInvoc(newArg)

	return &newArg
}

// goConvInvoc returns the code for converting from the flagArg to the goArg,
// either via a simple flagTypeConversion or via JSON unmarshalling
func goConvInvoc(a ClientArg) string {
	jsonConvTmpl := `
var {{.GoArg}} {{.GoType}}
if {{.FlagArg}} != nil && len(*{{.FlagArg}}) > 0 {
	err = json.Unmarshal([]byte(*{{.FlagArg}}), &{{.GoArg}})
	if err != nil {
		panic(errors.Wrapf(err, "unmarshalling {{.GoArg}} from %v:", {{.FlagArg}}))
	}
}
`
	if a.Repeated || !a.IsBaseType {
		code, err := applyTemplate("UnmarshalCliArgs", jsonConvTmpl, a, nil)
		if err != nil {
			panic(fmt.Sprintf("Couldn't apply template: %v", err))
		}
		return code
	}
	return fmt.Sprintf(`%s := %s`, a.GoArg, flagTypeConversion(a))
}

// createFlagConvertFunc creates the go string for the flag invocation to parse
// a command line argument into it's nearest available type that the flag
// package provides.
func createFlagConvertFunc(a ClientArg, methName string) string {
	fType := ""
	switch {
	case strings.Contains(a.FlagType, "uint32"):
		fType = `%s = fs%s.Uint("%s", 0, %s)`
	case strings.Contains(a.FlagType, "uint64"):
		fType = `%s = fs%s.Uint64("%s", 0, %s)`
	case strings.Contains(a.FlagType, "int32"):
		fType = `%s = fs%s.Int("%s", 0, %s)`
	case strings.Contains(a.FlagType, "int64"):
		fType = `%s = fs%s.Int64("%s", 0, %s)`
	case strings.Contains(a.FlagType, "bool"):
		fType = `%s = fs%s.Bool("%s", false, %s)`
	case strings.Contains(a.FlagType, "float32"):
		fType = `%s = fs%s.Float64("%s", 0.0, %s)`
	case strings.Contains(a.FlagType, "float64"):
		fType = `%s = fs%s.Float64("%s", 0.0, %s)`
	case strings.Contains(a.FlagType, "string"):
		fType = `%s = fs%s.String("%s", "", %s)`
	}

	return fmt.Sprintf(fType, a.FlagArg, methName, a.FlagName, `""`)
}

// flagTypeConversion creates the proper syntax for converting a flag into
// it's correct type. This is done because not every go type that a method
// field could be has a cooresponding flag command type. So this stage must
// exist to convert the subset of types which the flag package provides into
// other golang types, and the dereferencing is just a side effect of that.
func flagTypeConversion(a ClientArg) string {
	fType := ""
	switch {
	case strings.Contains(a.FlagType, "uint32"):
		fType = "uint32(*%s)"
	case strings.Contains(a.FlagType, "int32"):
		fType = "int32(*%s)"
	case strings.Contains(a.FlagType, "float32"):
		fType = "float32(*%s)"
	default:
		fType = "*%s"
	}
	return fmt.Sprintf(fType, a.FlagArg)
}

// applyTemplate applies a template with a given name, executor context, and
// function map. Returns the output of the template on success, returns an
// error if template failed to execute.
func applyTemplate(name string, tmpl string, executor interface{}, fncs template.FuncMap) (string, error) {
	codeTemplate := template.Must(template.New(name).Funcs(fncs).Parse(tmpl))

	code := bytes.NewBuffer(nil)
	err := codeTemplate.Execute(code, executor)
	if err != nil {
		return "", errors.Wrapf(err, "attempting to execute template %q", name)
	}
	return code.String(), nil
}

// lowCamelName returns a CamelCased string, but with the first letter
// lowercased. "example_name" becomes "exampleName".
func lowCamelName(s string) string {
	s = gogen.CamelCase(s)
	new := []rune(s)
	if len(new) < 1 {
		return s
	}
	rv := []rune{}
	rv = append(rv, unicode.ToLower(new[0]))
	rv = append(rv, new[1:]...)
	return string(rv)
}
