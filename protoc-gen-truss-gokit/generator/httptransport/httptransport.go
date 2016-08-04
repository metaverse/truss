// Package httptransport provides functions and template helpers for templating
// the http-transport of a go-kit based microservice.
package httptransport

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/TuneLab/gob/gendoc/doctree"
	"github.com/TuneLab/gob/protoc-gen-truss-gokit/generator/clientarggen"
	gogen "github.com/golang/protobuf/protoc-gen-go/generator"
)

type Helper struct {
	Methods []*Method
}

// NewHelper builds a helper struct from a service declaration.
func NewHelper(svc *doctree.ProtoService) *Helper {
	rv := Helper{}
	for _, meth := range svc.Methods {
		nMeth := Method{}
		nMeth.Name = meth.GetName()
		for i, binding := range meth.HttpBindings {
			nBinding := Binding{
				Label:        nMeth.Name + EnglishNumber(i),
				PathTemplate: binding.Path,
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
				nBinding.Fields = append(nBinding.Fields, &nField)
			}
			nMeth.Bindings = append(nMeth.Bindings, &nBinding)
		}
		rv.Methods = append(rv.Methods, &nMeth)
	}
	return &rv
}

func getParam(name string, params []*doctree.HttpParameter) *doctree.HttpParameter {
	for _, p := range params {
		if p.GetName() == name {
			return p
		}
	}
	return nil
}

// PathExtract takes a url and a gRPC-annotation style url template, and
// returns a map of the named parameters in the template and their values in
// the given url.
//
// PathExtract does not support the entirety of the URL template syntax defined
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
