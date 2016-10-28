package gengokit

import (
	"bytes"
	"go/format"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	log "github.com/Sirupsen/logrus"
	generatego "github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/pkg/errors"

	"github.com/TuneLab/go-truss/gengokit/astmodifier"
	"github.com/TuneLab/go-truss/gengokit/clientarggen"
	"github.com/TuneLab/go-truss/gengokit/httptransport"
	templateFileAssets "github.com/TuneLab/go-truss/gengokit/template"

	"github.com/TuneLab/go-truss/deftree"
	"github.com/TuneLab/go-truss/truss/truss"
)

func init() {
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stderr)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})
}

// templateExecutor is passed to templates as the executing struct; its fields
// and methods are used to modify the template
type templateExecutor struct {
	// import path for the directory containing the definition .proto files
	ImportPath string
	// import path for .pb.go files containing service structs
	PBImportPath string
	// PackageName is the name of the package containing the service definition
	PackageName string
	// GRPC/Protobuff service, with all parameters and return values accessible
	Service    *deftree.ProtoService
	ClientArgs *clientarggen.ClientServiceArgs
	// A helper struct for generating http transport functionality.
	HTTPHelper *httptransport.Helper
	funcMap    template.FuncMap
}

func newTemplateExecutor(dt deftree.Deftree, goPackage, goPBPackage string) (*templateExecutor, error) {
	service, err := getProtoService(dt)
	if err != nil {
		return nil, errors.Wrap(err, "no service found; aborting generating gokit service")
	}

	funcMap := template.FuncMap{
		"ToLower":    strings.ToLower,
		"Title":      strings.Title,
		"GoName":     generatego.CamelCase,
		"TrimPrefix": strings.TrimPrefix,
	}
	return &templateExecutor{
		ImportPath:   goPackage,
		PBImportPath: goPBPackage,
		PackageName:  dt.GetName(),
		Service:      service,
		ClientArgs:   clientarggen.New(service),
		HTTPHelper:   httptransport.NewHelper(service),
		funcMap:      funcMap,
	}, nil
}

// GenerateGokit returns a gokit service generated from a service definition (deftree),
// the package to the root of the generated service goPackage, the package
// to the .pb.go service struct files (goPBPackage) and any prevously generated files.
func GenerateGokit(dt deftree.Deftree, goPackage, goPBPackage string, previousFiles []truss.NamedReadWriter) ([]truss.NamedReadWriter, error) {
	te, err := newTemplateExecutor(dt, goPackage, goPBPackage)
	if err != nil {
		return nil, errors.Wrap(err, "could not create template executor")
	}

	fpm := make(map[string]io.Reader, len(previousFiles))
	for _, f := range previousFiles {
		fpm[f.Name()] = f
	}

	var codeGenFiles []truss.NamedReadWriter

	for _, templFP := range templateFileAssets.AssetNames() {
		file, err := generateResponseFile(templFP, te, fpm)
		if err != nil {
			return nil, errors.Wrap(err, "could not render template")
		}
		if file == nil {
			continue
		}

		codeGenFiles = append(codeGenFiles, file)
	}

	return codeGenFiles, nil
}

// getProtoService finds returns the service within a deftree.Deftree
func getProtoService(dt deftree.Deftree) (*deftree.ProtoService, error) {
	md := dt.(*deftree.MicroserviceDefinition)
	files := md.Files
	var service *deftree.ProtoService

	for _, file := range files {
		if len(file.Services) > 0 {
			service = file.Services[0]
		}
	}

	if service == nil {
		return nil, errors.New("no service found")
	}
	return service, nil
}

// generateResponseFile contains logic to choose how to render a template file
// based on path and if that file was generated previously. It accepts a
// template path to render, a templateExecutor to apply to the template, and a
// map of paths to files for the previous generation. It returns a
// truss.NamedReadWriter representing the generated file
func generateResponseFile(templFP string, te *templateExecutor, prevGenMap map[string]io.Reader) (truss.NamedReadWriter, error) {
	// Only .gotemplate files are rendered
	if filepath.Ext(templFP) != ".gotemplate" {
		log.WithField("Template file", templFP).Debug("Skipping rendering non-buildable partial template")
		return nil, nil
	}

	var genCode io.Reader
	var err error

	// Get the actual path to the file rather than the template file path
	actualFP := templatePathToActual(templFP, te.PackageName)

	// Map of template paths to template rendering functions
	renderFuncs := map[string]func(io.Reader, *templateExecutor) (io.Reader, error){
		"NAME-service/handlers/server/server_handler.gotemplate": updateServerMethods,
		"NAME-service/handlers/client/client_handler.gotemplate": updateClientMethods,
	}

	// If we are rendering a template with a renderFunc and the file existed
	// previously then use that function
	if renderFuncs[templFP] != nil {
		file := prevGenMap[actualFP]
		if file != nil {
			genCode, err = renderFuncs[templFP](file, te)
		}
	}

	if err != nil {
		return nil, errors.Wrap(err, "could not render template")
	}

	// if no code has been generated just apply the template
	if genCode == nil {
		genCode, err = applyTemplateFromPath(templFP, te)
	}

	if err != nil {
		return nil, errors.Wrap(err, "could not render template")
	}

	codeBytes, err := ioutil.ReadAll(genCode)
	if err != nil {
		return nil, err
	}

	// ignore error as we want to write the code either way to inspect after
	// writing to disk
	formattedCode := formatCode(codeBytes)

	var resp truss.SimpleFile

	// Set the path to the file and write the code to the file
	resp.Path = actualFP
	_, err = resp.Write(formattedCode)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// templatePathToActual accepts a templateFilePath and the packageName of the
// service and returns what the relative file path of what should be written to
// disk
func templatePathToActual(templFilePath, packageName string) string {
	// Switch "NAME" in path with packageName.
	// i.e. for packageName = addsvc; /NAME-service/NAME-server -> /addsvc-service/addsvc-server
	actual := strings.Replace(templFilePath, "NAME", packageName, -1)

	actual = strings.TrimSuffix(actual, "template")

	return actual
}

// updateServerMethods accepts NAME-service/handlers/server/server_handlers.go
// as an io.Reader and modifies it to contains only functions returned by
// serviceFunctionNames.
func updateServerMethods(svcHandler io.Reader, te *templateExecutor) (outCode io.Reader, err error) {
	svcFuncs := serviceFunctionsNames(te.Service.Methods)

	const svcMethodsTemplPath = "NAME-service/partial_template/service.methods"
	const svcInterfaceTemplPath = "NAME-service/partial_template/service.interface"

	astMod := astmodifier.New(svcHandler)

	astMod.RemoveFunctionsExecpt(svcFuncs)
	astMod.RemoveInterface("Service")

	// Index the handler functions, apply handler template for all function in
	// service definition that are not defined in handler
	currentFuncs := astMod.IndexFunctions()
	code := astMod.Buffer()

	trimmedTe := te.trimServiceFuncs(currentFuncs)

	newFuncs, err := applyTemplateFromPath(svcMethodsTemplPath, trimmedTe)
	if err != nil {
		return nil, errors.Wrap(err, "unable to apply service methods template")
	}

	code.ReadFrom(newFuncs)

	// Insert updated Service interface
	svcInterface, err := applyTemplateFromPath(svcInterfaceTemplPath, te)
	if err != nil {
		return nil, errors.Wrap(err, "unable to apply service interface template")
	}

	code.ReadFrom(svcInterface)

	return code, nil
}

// updateClientMethods accepts NAME-service/handlers/client/client_handler.go
// as an io.Reader and modifies it to contains only functions returned by
// serviceFunctionNames.
func updateClientMethods(clientHandler io.Reader, te *templateExecutor) (outCode io.Reader, err error) {
	svcFuncs := serviceFunctionsNames(te.Service.Methods)

	const clientMethodsTemplPath = "NAME-service/partial_template/client_handler.methods"

	astMod := astmodifier.New(clientHandler)

	// Remove functions no longer in definition
	astMod.RemoveFunctionsExecpt(svcFuncs)

	// Index handler functions, apply handler template for all function in
	// service definition that are not defined in handler
	currentFuncs := astMod.IndexFunctions()
	code := astMod.Buffer()

	// Get a templateExecutor that is identical but with only functions in the
	// definition file and not in the previously generated file
	trimmedTe := te.trimServiceFuncs(currentFuncs)

	newFuncs, err := applyTemplateFromPath(clientMethodsTemplPath, trimmedTe)
	if err != nil {
		return nil, errors.Wrap(err, "unable to apply client methods template")
	}

	code.ReadFrom(newFuncs)

	return code, nil
}

// serviceFunctionNames returns a slice of function names which are in the
// definition files plus the function "NewService". Used for inserting and
// removing functions from previously generated handler files
func serviceFunctionsNames(methods []*deftree.ServiceMethod) []string {
	var svcFuncs []string
	for _, m := range methods {
		svcFuncs = append(svcFuncs, m.GetName())
	}
	svcFuncs = append(svcFuncs, "NewService")

	return svcFuncs
}

// trimServiceFuncs removes functions in funcsInFile from the
// templateExecutor and returns a pointer to a new templateExecutor
func (te templateExecutor) trimServiceFuncs(funcsInFile map[string]bool) *templateExecutor {
	var methodsToTemplate []*deftree.ServiceMethod

	for _, m := range te.Service.Methods {
		mName := m.GetName()

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

// applyTemplateFromPath calls applyTemplate with the template at templFilePath
func applyTemplateFromPath(templFilePath string, executor *templateExecutor) (io.Reader, error) {

	templBytes, err := templateFileAssets.Asset(templFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to find template file: %v", templFilePath)
	}

	return applyTemplate(templBytes, templFilePath, executor)
}

func applyTemplate(templBytes []byte, templName string, executor *templateExecutor) (io.Reader, error) {
	templateString := string(templBytes)

	codeTemplate, err := template.New(templName).Funcs(executor.funcMap).Parse(templateString)
	if err != nil {
		return nil, errors.Wrap(err, "could not create template")
	}

	outputBuffer := bytes.NewBuffer(nil)
	err = codeTemplate.Execute(outputBuffer, executor)
	if err != nil {
		return nil, errors.Wrap(err, "template error")
	}

	return outputBuffer, nil
}

// formatCode takes a string representing golang code and attempts to return a
// formated copy of that code.  If formatting fails, a warning is logged and
// the original code is returned.
func formatCode(code []byte) []byte {
	formatted, err := format.Source(code)

	if err != nil {
		log.WithError(err).Warn("Code formatting error, generated service will not build, outputting unformatted code")
		// return code so at least we get something to examine
		return code
	}

	return formatted
}
