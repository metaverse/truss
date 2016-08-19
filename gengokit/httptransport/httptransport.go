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

	"github.com/TuneLab/go-truss/gendoc/doctree"
	"github.com/TuneLab/go-truss/gengokit/clientarggen"
	gogen "github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/pkg/errors"
)

// Helper is the base struct for the data structure containing all the
// information necessary to correctly template the HTTP transport functionality
// of a microservice. Helper must be built from a doctree.
type Helper struct {
	Methods           []*Method
	PathParamsBuilder string
}

// NewHelper builds a helper struct from a service declaration. The other
// "New*" functions in this file are there to make this function smaller and
// more testable.
func NewHelper(svc *doctree.ProtoService) *Helper {
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

// NewMethod builds a Method struct from a doctree.ServiceMethod.
func NewMethod(meth *doctree.ServiceMethod) *Method {
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

// NewBinding creates a Binding struct based on a doctree.HttpBinding. Because
// NewBinding requires access to some of it's parent method's fields, instead
// of passing a doctree.HttpBinding directly, you instead pass a
// doctree.ServiceMethod and the index of the HttpBinding within that methods
// "HttpBindings" slice.
func NewBinding(i int, meth *doctree.ServiceMethod) *Binding {
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
		if gt, ok = tmap[nField.ProtobufType]; !ok || field.Label == "LABEL_REPEATED" {
			gt = "string"
			nField.IsBaseType = false
		} else {
			nField.IsBaseType = true
		}
		nField.GoType = gt
		nField.ConvertFunc = createDecodeConvertFunc(nField)

		nField.CamelName = gogen.CamelCase(nField.Name)
		nField.LowCamelName = LowCamelName(nField.Name)

		nBinding.Fields = append(nBinding.Fields, &nField)
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

// createDecodeConvertFunc creates a go string representing the function to
// convert the string form of the field to it's correct go type.
func createDecodeConvertFunc(f Field) string {
	fType := ""
	switch {
	case strings.Contains(f.GoType, "int32"):
		fType = "%s, err := strconv.ParseInt(%s, 10, 32)"
	case strings.Contains(f.GoType, "int64"):
		fType = "%s, err := strconv.ParseInt(%s, 10, 64)"
	case strings.Contains(f.GoType, "int"):
		fType = "%s, err := strconv.ParseInt(%s, 10, 32)"
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

// The 'basePath' of a path is the section from the start of the string till
// the first '{' character.
func basePath(path string) string {
	parts := strings.Split(path, "{")
	return parts[0]
}

// getParam searches the slice of params for one named `name`, returning the
// first it finds. If no params have the given name, returns nil.
func getParam(name string, params []*doctree.HttpParameter) *doctree.HttpParameter {
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
