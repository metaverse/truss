package gengokit

import (
	"bytes"
	"io"
	"strings"
	"text/template"

	generatego "github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/pkg/errors"

	"github.com/TuneLab/go-truss/gengokit/clientarggen"
	"github.com/TuneLab/go-truss/gengokit/httptransport"
	"github.com/TuneLab/go-truss/svcdef"
	"github.com/TuneLab/go-truss/truss"
)

type Renderable interface {
	Render(string, *Executor) (io.Reader, error)
}

type Config struct {
	GoPackage string
	PBPackage string

	PreviousFiles []truss.NamedReadWriter
}

// Executor is passed to templates as the executing struct; its fields
// and methods are used to modify the template
type Executor struct {
	// import path for the directory containing the definition .proto files
	ImportPath string
	// import path for .pb.go files containing service structs
	PBImportPath string
	// PackageName is the name of the package containing the service definition
	PackageName string
	// GRPC/Protobuff service, with all parameters and return values accessible
	Service    *svcdef.Service
	ClientArgs *clientarggen.ClientServiceArgs
	// A helper struct for generating http transport functionality.
	HTTPHelper *httptransport.Helper
	FuncMap    template.FuncMap
}

func NewExecutor(sd *svcdef.Svcdef, conf Config) (*Executor, error) {
	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
		"GoName":  generatego.CamelCase,
	}
	return &Executor{
		ImportPath:   conf.GoPackage,
		PBImportPath: conf.PBPackage,
		PackageName:  sd.PkgName,
		Service:      sd.Service,
		ClientArgs:   clientarggen.New(sd.Service),
		HTTPHelper:   httptransport.NewHelper(sd.Service),
		FuncMap:      funcMap,
	}, nil
}

// ApplyTemplate applies the passed template with the Executor
func (e *Executor) ApplyTemplate(templ string, templName string) (io.Reader, error) {
	return ApplyTemplate(templ, templName, e, e.FuncMap)
}

// ApplyTemplate is a helper methods that packages can call to render a
// template with any executor and func map
func ApplyTemplate(templ string, templName string, executor interface{}, funcMap template.FuncMap) (io.Reader, error) {
	codeTemplate, err := template.New(templName).Funcs(funcMap).Parse(templ)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create template")
	}

	outputBuffer := bytes.NewBuffer(nil)
	err = codeTemplate.Execute(outputBuffer, executor)
	if err != nil {
		return nil, errors.Wrap(err, "template error")
	}

	return outputBuffer, nil
}
