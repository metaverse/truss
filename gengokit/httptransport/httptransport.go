// Package httptransport provides functions and template helpers for templating
// the http-transport of a go-kit based microservice.
package httptransport

import (
	"bytes"
	"fmt"
	"go/format"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	log "github.com/Sirupsen/logrus"
	"github.com/TuneLab/go-truss/deftree"
	"github.com/TuneLab/go-truss/gengokit/clientarggen"
	gogen "github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/pkg/errors"
)

// Helper is the base struct for the data structure containing all the
// information necessary to correctly template the HTTP transport functionality
// of a microservice. Helper must be built from a deftree.
type Helper struct {
	Methods           []*Method
	PathParamsBuilder string
}

// NewHelper builds a helper struct from a service declaration. The other
// "New*" functions in this file are there to make this function smaller and
// more testable.
func NewHelper(svc *deftree.ProtoService) *Helper {
	// The HTTPAssistFuncs global is a group of function literals defined
	// within templates.go
	pp := FormatCode(HTTPAssistFuncs)
	rv := Helper{
		PathParamsBuilder: pp,
	}
	for _, meth := range svc.Methods {
		nMeth := NewMethod(meth)
		rv.Methods = append(rv.Methods, nMeth)
	}
	return &rv
}

// NewMethod builds a Method struct from a deftree.ServiceMethod.
func NewMethod(meth *deftree.ServiceMethod) *Method {
	nMeth := Method{
		Name:         meth.GetName(),
		RequestType:  meth.RequestType.GetName(),
		ResponseType: meth.ResponseType.GetName(),
	}
	for i := range meth.HttpBindings {
		nBinding := NewBinding(i, meth)
		nBinding.Parent = &nMeth
		nMeth.Bindings = append(nMeth.Bindings, nBinding)
	}
	return &nMeth
}

// NewBinding creates a Binding struct based on a deftree.HttpBinding. Because
// NewBinding requires access to some of it's parent method's fields, instead
// of passing a deftree.HttpBinding directly, you instead pass a
// deftree.ServiceMethod and the index of the HttpBinding within that methods
// "HttpBindings" slice.
func NewBinding(i int, meth *deftree.ServiceMethod) *Binding {
	binding := meth.HttpBindings[i]
	nBinding := Binding{
		Label:        meth.GetName() + EnglishNumber(i),
		PathTemplate: binding.Path,
		BasePath:     basePath(binding.Path),
		Verb:         binding.Verb,
	}
	for _, field := range meth.RequestType.Fields {
		// Param is specifically an http parameter, while field is a
		// field in a protobuf msg. nField is a distillation of the
		// relevant information to translate the http parameter into a
		// field on a protobuf msg.
		param := getParam(field.GetName(), binding.Params)
		// TODO add handling for non-found params here
		nField := Field{
			Name:          field.GetName(),
			Location:      param.Location,
			ProtobufType:  param.Type,
			ProtobufLabel: field.Label,
			LocalName:     fmt.Sprintf("%s%s", gogen.CamelCase(field.GetName()), gogen.CamelCase(meth.GetName())),
		}
		var gt string
		var ok bool
		tmap := clientarggen.ProtoToGoTypeMap
		if gt, ok = tmap[nField.ProtobufType]; !ok {
			gt = "string"
			nField.IsBaseType = false
		} else {
			nField.IsBaseType = true
		}
		nField.GoType = gt
		nField.ConvertFunc = createDecodeConvertFunc(nField)
		nField.TypeConversion = createDecodeTypeConversion(nField)

		nField.CamelName = gogen.CamelCase(nField.Name)
		nField.LowCamelName = LowCamelName(nField.Name)

		nBinding.Fields = append(nBinding.Fields, &nField)

		// Emit warnings for certain cases
		if !nField.IsBaseType {
			log.Warnf("%s.%s is a custom type '%s', only base types and repeated base "+
				"types are supported. As a result, the generated HTTP "+
				"transport will fail to compile. Remove non-base types.",
				meth.GetName(),
				nField.Name,
				nField.ProtobufType)
		}
		if field.Label == "LABEL_REPEATED" && nField.Location == "path" {
			log.Warnf(
				"%s.%s is a repeated field specified to be in the path. "+
					"Repeated fields are not supported in the path and may"+
					"result in generated code which fails to compile.",
				meth.GetName(),
				nField.Name)
		}
	}
	return &nBinding
}

// GenServerDecode returns the generated code for the server-side decoding of
// an http request into its request struct.
func (b *Binding) GenServerDecode() (string, error) {
	code, err := ApplyTemplate("ServerDecodeTemplate", ServerDecodeTemplate, b, TemplateFuncs)
	if err != nil {
		return "", err
	}
	code = FormatCode(code)
	return code, nil
}

// GenClientEncode returns the generated code for the client-side encoding of
// that clients request struct into the correctly formatted http request.
func (b *Binding) GenClientEncode() (string, error) {
	code, err := ApplyTemplate("ClientEncodeTemplate", ClientEncodeTemplate, b, TemplateFuncs)
	if err != nil {
		return "", err
	}
	code = FormatCode(code)
	return code, nil
}

// PathSections returns a slice of strings for templating the creation of a
// fully assembled URL with the correct fields in the correct locations.
//
// For example, let's say there's a method "Sum" which accepts a "SumRequest",
// and SumRequest has two fields, 'a' and 'b'. Additionally, lets say that this
// binding for "Sum" has a path of "/sum/{a}". If we call the PathSection()
// method on this binding, it will return a slice that looks like the
// following slice literal:
//
//     []string{
//         "\"\"",
//         "\"sum\"",
//         "fmt.Sprint(req.A)",
//     }
func (b *Binding) PathSections() []string {
	rv := []string{}
	parts := strings.Split(b.PathTemplate, "/")
	for _, part := range parts {
		if len(part) > 2 && part[0] == '{' && part[len(part)-1] == '}' {
			name := RemoveBraces(part)
			convert := fmt.Sprintf("fmt.Sprint(req.%v)", gogen.CamelCase(name))
			rv = append(rv, convert)
		} else {
			// Add quotes around things which'll be embeded as string literals,
			// so that the 'fmt.Sprint' lines will be unquoted and thus
			// evaluated as code.
			rv = append(rv, `"`+part+`"`)
		}
	}
	return rv
}

// GenQueryUnmarshaler returns the generated code for server-side unmarshaling
// of a query parameter into it's correct field on the request struct.
func (f *Field) GenQueryUnmarshaler() (string, error) {
	repeatedQueryLogic := `
for _, {{.LocalName}}Str := range r.URL.Query()["{{.Name}}"] {
	{{.ConvertFunc}}
	if err != nil {
		fmt.Printf("Error while extracting {{.LocalName}} from {{.Location}}: %v\n", err)
		fmt.Printf("{{.Location}}Params: %v\n", {{.Location}}Params)
		return nil, err
	}
	req.{{.CamelName}} = append(req.{{.CamelName}}, {{.TypeConversion}})
}
`
	genericLogic := `
{{.LocalName}}Str := {{.Location}}Params["{{.Name}}"]
{{.ConvertFunc}}
// TODO: Better error handling
if err != nil {
	fmt.Printf("Error while extracting {{.LocalName}} from {{.Location}}: %v\n", err)
	fmt.Printf("{{.Location}}Params: %v\n", {{.Location}}Params)
	return nil, err
}
req.{{.CamelName}} = {{.TypeConversion}}
`
	var selected string
	if f.Location == "query" && f.ProtobufLabel == "LABEL_REPEATED" {
		selected = repeatedQueryLogic
	} else if f.Location != "body" {
		selected = genericLogic
	}
	code, err := ApplyTemplate("FieldEncodeLogic", selected, f, TemplateFuncs)
	if err != nil {
		return "", err
	}
	code = FormatCode(code)
	return code, nil
}

// createDecodeConvertFunc creates a go string representing the function to
// convert the string form of the field to it's correct go type.
func createDecodeConvertFunc(f Field) string {
	fType := ""
	switch {
	case strings.Contains(f.GoType, "uint32"):
		fType = "%s, err := strconv.ParseUint(%s, 10, 32)"
	case strings.Contains(f.GoType, "uint64"):
		fType = "%s, err := strconv.ParseUint(%s, 10, 64)"
	case strings.Contains(f.GoType, "int32"):
		fType = "%s, err := strconv.ParseInt(%s, 10, 32)"
	case strings.Contains(f.GoType, "int64"):
		fType = "%s, err := strconv.ParseInt(%s, 10, 64)"
	case strings.Contains(f.GoType, "bool"):
		fType = "%s, err := strconv.ParseBool(%s)"
	case strings.Contains(f.GoType, "float32"):
		fType = "%s, err := strconv.ParseFloat(%s, 32)"
	case strings.Contains(f.GoType, "float64"):
		fType = "%s, err := strconv.ParseFloat(%s, 64)"
	case strings.Contains(f.GoType, "string"):
		fType = "%s := %s"
	}
	return fmt.Sprintf(fType, f.LocalName, f.LocalName+"Str")
}

// createDecodeTypeConversion creates a go string that converts a 64 bit type to a 32 bit type
// as strconv.ParseInt, ParseUInt, and ParseFloat always return the 64 bit type
func createDecodeTypeConversion(f Field) string {
	fType := ""
	switch {
	case strings.Contains(f.GoType, "uint32"):
		fType = "uint32(%s)"
	case strings.Contains(f.GoType, "int32"):
		fType = "int32(%s)"
	case strings.Contains(f.GoType, "float32"):
		fType = "float32(%s)"
	default:
		fType = "%s"
	}
	return fmt.Sprintf(fType, f.LocalName)
}

// The 'basePath' of a path is the section from the start of the string till
// the first '{' character.
func basePath(path string) string {
	parts := strings.Split(path, "{")
	return parts[0]
}

// getParam searches the slice of params for one named `name`, returning the
// first it finds. If no params have the given name, returns nil.
func getParam(name string, params []*deftree.HttpParameter) *deftree.HttpParameter {
	for _, p := range params {
		if p.GetName() == name {
			return p
		}
	}
	return nil
}

// DigitEnglish is a map of runes of digits zero to nine to their lowercase
// english language spellings.
var DigitEnglish = map[rune]string{
	'0': "zero",
	'1': "one",
	'2': "two",
	'3': "three",
	'4': "four",
	'5': "five",
	'6': "six",
	'7': "seven",
	'8': "eight",
	'9': "nine",
}

// EnglishNumber takes an integer and returns the english words that represents
// that number, in base ten. Examples:
//     1  -> "One"
//     5  -> "Five"
//     10 -> "OneZero"
//     48 -> "FourEight"
func EnglishNumber(i int) string {
	n := strconv.Itoa(i)
	rv := ""
	for _, c := range n {
		if engl, ok := DigitEnglish[rune(c)]; ok {
			rv += strings.Title(engl)
		}
	}
	return rv
}

// LowCamelName returns a CamelCased string, but with the first letter
// lowercased. "package_name" becomes "packageName".
func LowCamelName(s string) string {
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

// TemplateFuncs contains a series of utility functions to be passed into
// templates and used within those templates.
var TemplateFuncs = template.FuncMap{
	"ToLower": strings.ToLower,
	"Title":   strings.Title,
	"GoName":  gogen.CamelCase,
}

// ApplyTemplate applies a template with a given name, executor context, and
// function map. Returns the output of the template on success, returns an
// error if template failed to execute.
func ApplyTemplate(name string, tmpl string, executor interface{}, fncs template.FuncMap) (string, error) {
	codeTemplate := template.Must(template.New(name).Funcs(fncs).Parse(tmpl))

	code := bytes.NewBuffer(nil)
	err := codeTemplate.Execute(code, executor)
	if err != nil {
		return "", errors.Wrapf(err, "attempting to execute template %q", name)
	}
	return code.String(), nil
}

// FormatCode takes a string representing some go code and attempts to format
// that code. If formating fails, the original source code is returned.
func FormatCode(code string) string {
	formatted, err := format.Source([]byte(code))

	if err != nil {
		// Set formatted to code so at least we get something to examine
		formatted = []byte(code)
	}

	return string(formatted)
}
