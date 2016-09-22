// Package clientarggen collects information for templating the code in a
// truss-generated client which marshals command line flags into message fields
// for each service. Functions and fields in clientargen are called by
// templates in protoc-gen-truss-gokit/template/
package clientarggen

import (
	"fmt"
	"strings"

	"github.com/TuneLab/go-truss/deftree"
	generatego "github.com/golang/protobuf/protoc-gen-go/generator"
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
	Name string

	FlagName        string
	FlagArg         string
	FlagType        string
	FlagConvertFunc string

	GoArg          string
	GoType         string
	GoConvertInvoc string
	GoConvertFunc  string

	ProtbufType string

	IsBaseType bool
	Repeated   bool
}

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
func (self *MethodArgs) FunctionArgs() string {
	tmp := []string{}
	for _, a := range self.Args {
		tmp = append(tmp, fmt.Sprintf("%s %s", a.FlagArg, a.FlagType))
	}
	return strings.Join(tmp, ", ")
}

// CallArgs returns a string for the variables to pass to a function
// implementing this method. Example:
//
//     request, _ := clientHandler.Sum(ASum,  BSum)
//                                     └──────────┘
func (self *MethodArgs) CallArgs() string {
	tmp := []string{}
	for _, a := range self.Args {
		tmp = append(tmp, createFlagConversion(*a))
	}
	return strings.Join(tmp, ", ")
}

type ClientServiceArgs struct {
	MethArgs map[string]*MethodArgs
}

// AllFlags returns a string that is all the flag declarations for all
// arguments of all methods, separated by newlines. This is used in the
// template to declare all the flag arguments for a client at once, and without
// doing all this iteration in a template where it would be much less
// understandable.
func (self *ClientServiceArgs) AllFlags() string {
	tmp := []string{}
	for _, m := range self.MethArgs {
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
func New(svc *deftree.ProtoService) *ClientServiceArgs {
	svcArgs := ClientServiceArgs{
		MethArgs: make(map[string]*MethodArgs),
	}
	for _, meth := range svc.Methods {
		m := MethodArgs{}
		for _, field := range meth.RequestType.Fields {
			newArg := newClientArg(meth.GetName(), field)
			m.Args = append(m.Args, newArg)
		}
		svcArgs.MethArgs[meth.GetName()] = &m
	}

	return &svcArgs
}

func newClientArg(methName string, field *deftree.MessageField) *ClientArg {
	newArg := ClientArg{}
	newArg.Name = field.GetName()

	if field.Label == "LABEL_REPEATED" {
		newArg.Repeated = true
	}
	newArg.ProtbufType = field.Type.GetName()

	newArg.FlagName = fmt.Sprintf("%s.%s", strings.ToLower(methName), strings.ToLower(field.GetName()))
	newArg.FlagArg = fmt.Sprintf("flag%s%s", generatego.CamelCase(newArg.Name), generatego.CamelCase(methName))

	var ft string
	var ok bool
	// For types outside the base types, have flag treat them as strings
	if ft, ok = ProtoToGoTypeMap[field.Type.GetName()]; !ok {
		ft = "string"
		newArg.IsBaseType = false
	} else {
		newArg.IsBaseType = true
	}
	if newArg.Repeated {
		ft = "string"
	}
	newArg.FlagType = ft
	newArg.FlagConvertFunc = createFlagConvertFunc(newArg)

	newArg.GoArg = fmt.Sprintf("%s%s", generatego.CamelCase(newArg.Name), generatego.CamelCase(methName))
	// For types outside the base types, treat them as strings
	if newArg.IsBaseType {
		newArg.GoType = ProtoToGoTypeMap[field.Type.GetName()]
	} else {
		newArg.GoType = "string"
	}

	newArg.GoConvertFunc = GenerateCarveFunc(&newArg)
	newArg.GoConvertInvoc = GenerateCarveInvocation(&newArg)

	return &newArg
}

// createFlagConvertFunc creates the go string for the flag invocation to parse
// a command line argument into it's nearest available type that the flag
// package provides.
func createFlagConvertFunc(a ClientArg) string {
	fType := ""
	switch {
	case strings.Contains(a.FlagType, "uint32"):
		fType = `%s = flag.Uint("%s", 0, %s)`
	case strings.Contains(a.FlagType, "uint64"):
		fType = `%s = flag.Uint64("%s", 0, %s)`
	case strings.Contains(a.FlagType, "int32"):
		fType = `%s = flag.Int("%s", 0, %s)`
	case strings.Contains(a.FlagType, "int64"):
		fType = `%s = flag.Int64("%s", 0, %s)`
	case strings.Contains(a.FlagType, "bool"):
		fType = `%s = flag.Bool("%s", false, %s)`
	case strings.Contains(a.FlagType, "float32"):
		fType = `%s = flag.Float64("%s", 0.0, %s)`
	case strings.Contains(a.FlagType, "float64"):
		fType = `%s = flag.Float64("%s", 0.0, %s)`
	case strings.Contains(a.FlagType, "string"):
		fType = `%s = flag.String("%s", "", %s)`
	}
	return fmt.Sprintf(fType, a.FlagArg, a.FlagName, `""`)
}

// createFlagConversion creates the proper syntax for converting a flag into
// it's correct type. This is done because not every go type that a method
// field could be has a cooresponding flag command type. So this stage must
// exist to convert the subset of types which the flag package provides into
// other golang types, and the dereferencing is just a side effect of that.
func createFlagConversion(a ClientArg) string {
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
	return fmt.Sprintf(fType, generatego.CamelCase(a.FlagArg))
}
