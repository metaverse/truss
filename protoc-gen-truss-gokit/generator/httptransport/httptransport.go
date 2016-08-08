// Package httptransport provides functions and template helpers for templating
// the http-transport of a go-kit based microservice.
package httptransport

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/TuneLab/gob/gendoc/doctree"
	"github.com/TuneLab/gob/protoc-gen-truss-gokit/generator/clientarggen"
	gogen "github.com/golang/protobuf/protoc-gen-go/generator"
)

type Helper struct {
	Methods           []*Method
	PathParamsBuilder string
	//QueryParamsBuilder string
}

// NewHelper builds a helper struct from a service declaration. The other
// "New*" functions in this file are there to make this function smaller and
// more testable.
func NewHelper(svc *doctree.ProtoService) *Helper {
	pp, _ := AllFuncSourceCode(PathParams)
	rv := Helper{
		PathParamsBuilder: pp,
	}
	for _, meth := range svc.Methods {
		nMeth := NewMethod(meth)
		rv.Methods = append(rv.Methods, nMeth)
	}
	return &rv
}

func NewMethod(meth *doctree.ServiceMethod) *Method {
	nMeth := Method{
		Name:        meth.GetName(),
		RequestType: meth.RequestType.GetName(),
	}
	for i, _ := range meth.HttpBindings {
		nBinding := NewBinding(i, meth)
		nMeth.Bindings = append(nMeth.Bindings, nBinding)
	}
	return &nMeth
}

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

func (self *Binding) PathSections() []string {
	rv := []string{}
	parts := strings.Split(self.PathTemplate, "/")
	for _, part := range parts {
		if len(part) > 2 && part[0] == '{' && part[len(part)-1] == '}' {
			name := RemoveBraces(part)
			convert := fmt.Sprintf("fmt.Sprint(req.%v)", gogen.CamelCase(name))
			rv = append(rv, convert)
		} else {
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
// that number, in base ten
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

// LowCamelCase returns a CamelCased string, but with the first letter
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
