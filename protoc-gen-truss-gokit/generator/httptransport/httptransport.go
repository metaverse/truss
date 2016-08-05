// Package httptransport provides functions and template helpers for templating
// the http-transport of a go-kit based microservice.
package httptransport

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/TuneLab/gob/gendoc/doctree"
	"github.com/TuneLab/gob/protoc-gen-truss-gokit/generator/clientarggen"
	gogen "github.com/golang/protobuf/protoc-gen-go/generator"
)

type Helper struct {
	Methods            []*Method
	PathParamsBuilder  string
	QueryParamsBuilder string
	PossibleLocations  []string
}

// NewHelper builds a helper struct from a service declaration. The other
// "New*" functions in this file are there to make this function smaller and
// more testable.
func NewHelper(svc *doctree.ProtoService) *Helper {
	pp, _ := GetSourceCode(PathParams)
	qp, _ := GetSourceCode(QueryParams)
	rv := Helper{
		PathParamsBuilder:  pp,
		QueryParamsBuilder: qp,
		PossibleLocations:  []string{"query", "body", "path"},
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
		nField.ConvertFunc = createConvertFunc(nField)

		nBinding.Fields = append(nBinding.Fields, &nField)
	}
	return &nBinding
}

// createConvertFunc creates a go string representing the function to convert
// the string form of the field to it's correct go type.
func createConvertFunc(f Field) string {
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

func basePath(path string) string {
	parts := strings.Split(path, "{")
	return parts[0]
}

func getParam(name string, params []*doctree.HttpParameter) *doctree.HttpParameter {
	for _, p := range params {
		if p.GetName() == name {
			return p
		}
	}
	return nil
}

// PathParams takes a url and a gRPC-annotation style url template, and
// returns a map of the named parameters in the template and their values in
// the given url.
//
// PathParams does not support the entirety of the URL template syntax defined
// in third_party/googleapis/google/api/httprule.proto. Only a small subset of
// the functionality defined there is implemented here.
func PathParams(url string, urlTmpl string) (map[string]string, error) {
	removeBraces := func(val string) string {
		val = strings.Replace(val, "{", "", -1)
		val = strings.Replace(val, "}", "", -1)
		return val
	}
	buildParamMap := func(urlTmpl string) map[string]int {
		rv := map[string]int{}

		parts := strings.Split(urlTmpl, "/")
		for idx, part := range parts {
			if strings.ContainsAny(part, "{}") {
				param := removeBraces(part)
				rv[param] = idx
			}
		}
		return rv
	}
	rv := map[string]string{}
	pmp := buildParamMap(urlTmpl)

	parts := strings.Split(url, "/")
	for k, v := range pmp {
		rv[k] = parts[v]
	}

	return rv, nil
}

func QueryParams(vals url.Values) (map[string]string, error) {
	// TODO make this not flatten the query params
	// WARNING this is a super huge hack and will ignore repeated values in the
	// query parameter. This should absolutely be correctly implemented later
	// by someone else or maybe future me...
	rv := map[string]string{}
	for k, v := range vals {
		rv[k] = v[0]
	}
	return rv, nil
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
