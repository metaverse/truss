package gengokit

import (
	"io"
	"strings"
	"text/template"

	log "github.com/Sirupsen/logrus"
	generatego "github.com/golang/protobuf/protoc-gen-go/generator"

	"github.com/TuneLab/go-truss/gengokit/clientarggen"
	"github.com/TuneLab/go-truss/gengokit/httptransport"
	"github.com/TuneLab/go-truss/svcdef"
	"github.com/TuneLab/go-truss/truss/truss"
)

type Renderable interface {
	Render(string, *TemplateExecutor) (io.Reader, error)
}

type Config struct {
	GoPackage string
	PBPackage string

	PreviousFiles []truss.NamedReadWriter
}

// templateExecutor is passed to templates as the executing struct; its fields
// and methods are used to modify the template
type TemplateExecutor struct {
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

func NewTemplateExecutor(sd *svcdef.Svcdef, conf Config) (*TemplateExecutor, error) {
	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
		"GoName":  generatego.CamelCase,
	}
	return &TemplateExecutor{
		ImportPath:   conf.GoPackage,
		PBImportPath: conf.PBPackage,
		PackageName:  sd.PkgName,
		Service:      sd.Service,
		ClientArgs:   clientarggen.New(sd.Service),
		HTTPHelper:   httptransport.NewHelper(sd.Service),
		FuncMap:      funcMap,
	}, nil
}

// trimServiceFuncs removes functions in funcsInFile from the
// templateExecutor and returns a pointer to a new templateExecutor
func (te TemplateExecutor) TrimServiceFuncs(funcsInFile map[string]bool) *TemplateExecutor {
	var methodsToTemplate []*svcdef.ServiceMethod

	for _, m := range te.Service.Methods {
		mName := m.Name

		if funcsInFile[mName] {
			log.WithField("Method", mName).Info("Handler method already exists")
			continue
		}
		methodsToTemplate = append(methodsToTemplate, m)
		log.WithField("Method", mName).Info("Rendering template for method")
	}

	// templateExec's Service is dereference and that new Service's
	// pointer to its messages is changed to be methodsToTemplate
	tempService := *te.Service
	tempService.Methods = methodsToTemplate

	te.Service = &tempService

	return &te
}
