// Package clientarggen collects the necessary information for templating the
// business logic of a truss client. Let's look at an example.
//
// Say the user wants to create a service that adds two numbers together. That
// user has created a protobuf file which looks like this:
//
//     syntax = "proto3";
//     package pb;
//     import "google/api/annotations.proto";
//
//     // The Add service definition.
//     service Add {
//       // Sums two integers.
//       rpc Sum (SumRequest) returns (SumReply) {
//         option (google.api.http) = {
//           get: "/sum"
//         };
//       }
//     }
//
//     // The sum request contains two parameters.
//     message SumRequest {
//       int64 a = 1;
//       int64 b = 2;
//     }
//
//     // The sum response contains the result of the calculation.
//     message SumReply {
//       int64 v = 1;
//       string err = 2;
//     }
//
// We cannot infer what the business logic of this service is purely from the
// definition, so the part of the server where the numbers a and b are added
// together and returned must be implemented by the user.
//
// However, the command line client, which we generate, is much simpler. All it
// does is take some arguments on the command line, convert those to the
// correct types, place those arguments into a struct of the correct type, then
// make a request to the server. Determining the nature of each command line
// argument, how to convert it to the correct type, and other bookkeeping to
// automatically generate a client is the task of the package clientarggen.
//
// Note that the code generation portion of creating client business logic is
// largely done in the template files, but that those template files rely on
// information which is collected and made conveniently available ahead of time
// by this package.
package clientarggen

import (
	"fmt"
	"strings"

	//	log "github.com/Sirupsen/logrus"
	"github.com/TuneLab/gob/gendoc/doctree"
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
	Name            string
	FlagName        string
	FlagArg         string
	ProtbufType     string
	GoType          string
	FlagConvertFunc string
	IsBaseType      bool
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
		tmp = append(tmp, fmt.Sprintf("%s %s", a.FlagArg, a.GoType))
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
func New(svc *doctree.ProtoService) *ClientServiceArgs {
	svcArgs := ClientServiceArgs{
		MethArgs: make(map[string]*MethodArgs),
	}
	for _, meth := range svc.Methods {
		m := MethodArgs{}
		for _, field := range meth.RequestType.Fields {
			newArg := ClientArg{}
			newArg.Name = field.GetName()
			newArg.FlagName = fmt.Sprintf("%s.%s", strings.ToLower(meth.GetName()), strings.ToLower(field.GetName()))
			newArg.FlagArg = fmt.Sprintf("%s%s", generatego.CamelCase(newArg.Name), generatego.CamelCase(meth.GetName()))

			newArg.ProtbufType = field.Type.GetName()

			var gt string
			var ok bool
			if gt, ok = ProtoToGoTypeMap[field.Type.GetName()]; !ok || field.Label == "LABEL_REPEATED" {
				gt = "string"
				newArg.IsBaseType = false
			} else {
				newArg.IsBaseType = true
			}
			newArg.GoType = gt

			newArg.FlagConvertFunc = createFlagConvertFunc(newArg)

			m.Args = append(m.Args, &newArg)
		}
		svcArgs.MethArgs[meth.GetName()] = &m
	}

	return &svcArgs
}

// createFlagConvertFunc creates the go string for the flag invocation to parse
// a command line argument into it's nearest available type that the flag
// package provides.
func createFlagConvertFunc(a ClientArg) string {
	fType := ""
	switch {
	case strings.Contains(a.GoType, "int32"):
		fType = "%s = flag.Int(\"%s\", 0, %s)"
	case strings.Contains(a.GoType, "int64"):
		fType = "%s = flag.Int64(\"%s\", 0, %s)"
	case strings.Contains(a.GoType, "int"):
		fType = "%s = flag.Int(\"%s\", 0, %s)"
	case strings.Contains(a.GoType, "bool"):
		fType = "%s = flag.Bool(\"%s\", false, %s)"
	case strings.Contains(a.GoType, "float32"):
		fType = "%s = flag.Float64(\"%s\", 0.0, %s)"
	case strings.Contains(a.GoType, "float64"):
		fType = "%s = flag.Float64(\"%s\", 0.0, %s)"
	case strings.Contains(a.GoType, "string"):
		fType = "%s = flag.String(\"%s\", \"\", %s)"
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
	case strings.Contains(a.GoType, "int32"):
		fType = "int32(*%s)"
	case strings.Contains(a.GoType, "int64"):
		fType = "*%s"
	case strings.Contains(a.GoType, "int"):
		fType = "*%s"
	case strings.Contains(a.GoType, "bool"):
		fType = "*%s"
	case strings.Contains(a.GoType, "float32"):
		fType = "float32(*%s)"
	case strings.Contains(a.GoType, "float64"):
		fType = "*%s"
	case strings.Contains(a.GoType, "string"):
		fType = "*%s"
	}
	return fmt.Sprintf(fType, generatego.CamelCase(a.FlagArg))
}
