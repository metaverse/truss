package gengokit

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	log "github.com/Sirupsen/logrus"
	generatego "github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/pkg/errors"

	"github.com/TuneLab/gob/gengokit/astmodifier"
	"github.com/TuneLab/gob/gengokit/clientarggen"
	templateFileAssets "github.com/TuneLab/gob/gengokit/template"

	"github.com/TuneLab/gob/gendoc/doctree"
	"github.com/TuneLab/gob/truss/truss"
)

func init() {
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stderr)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})
}

type generator struct {
	previousFiles     []truss.SimpleFile
	templateFileNames func() []string
	templateFile      func(string) ([]byte, error)
	templateFuncMap   template.FuncMap
	templateExec      templateExecutor
}

// templateExecutor is passed to templates as the executing struct its fields
// and methods are used to modify the template
type templateExecutor struct {
	// Import path for handler package
	HandlerImport string
	// Import path for generated packages
	GeneratedImport string
	// GRPC/Protobuff service, with all parameters and return values accessible
	Service    *doctree.ProtoService
	ClientArgs *clientarggen.ClientServiceArgs
}

// GenerateGokit accepts a doctree representing the ast of a group of .proto
// files, a []truss.SimpleFile representing files generated previously, and
// a goImportPath for templating go code imports
// GenerateGoCode returns the a []truss.SimpleFile representing a generated
// gokit microservice file structure
func GenerateGokit(dt doctree.Doctree, previousFiles []truss.SimpleFile, goImportPath string) ([]truss.SimpleFile, error) {
	service, err := getProtoService(dt)
	if err != nil {
		return nil, errors.Wrap(err, "no service found aborting generating gokit microservice")
	}

	g := newGenerator(service, previousFiles, goImportPath)
	files, err := g.GenerateResponseFiles()

	if err != nil {
		return nil, errors.Wrap(err, "could not generate gokit microservice")
	}

	return files, nil
}

// getProtoService finds returns the service within a doctree.Doctree
func getProtoService(dt doctree.Doctree) (*doctree.ProtoService, error) {
	md := dt.(*doctree.MicroserviceDefinition)
	files := md.Files
	var service *doctree.ProtoService

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

// New returns a new generator which generates a gokit microservice
func newGenerator(service *doctree.ProtoService, previousFiles []truss.SimpleFile, goImportPath string) *generator {
	// import path for server and client handlers
	handlerImportString := goImportPath + "/service"
	// import path for generated code that user should not edit
	generatedImportString := handlerImportString + "/DONOTEDIT"

	funcMap := template.FuncMap{
		"ToLower":    strings.ToLower,
		"Title":      strings.Title,
		"GoName":     generatego.CamelCase,
		"TrimPrefix": strings.TrimPrefix,
	}

	return &generator{
		previousFiles:     previousFiles,
		templateFileNames: templateFileAssets.AssetNames,
		templateFile:      templateFileAssets.Asset,
		templateFuncMap:   funcMap,
		templateExec: templateExecutor{
			HandlerImport:   handlerImportString,
			GeneratedImport: generatedImportString,
			Service:         service,
			ClientArgs:      clientarggen.New(service),
		},
	}
}

// getFileContentByName searches though a []truss.SimpleFile and returns the
// contents of the file with the name n
func getFileContentByName(n string, files []truss.SimpleFile) *string {
	for _, f := range files {
		if *(f.Name) == n {
			return f.Content
		}
	}
	return nil
}

// updateServiceMethods will update the functions within an existing service.go
// file so it contains all the functions within svcFuncs and ONLY those
// functions within svcFuncs
func (g *generator) updateServiceMethods(svcHandler *string, svcFuncs []string) (outCode *bytes.Buffer, err error) {
	const svcMethodsTemplPath = "service/partial_template/service.methods"
	const svcInterfaceTemplPath = "service/partial_template/service.interface"

	astMod := astmodifier.New(svcHandler)

	//TODO: Discuss if functions should be removed from the service file, when using truss I did not like that it removed function I wrote myself
	//astMod.RemoveFunctionsExecpt(svcFuncs)
	astMod.RemoveInterface("Service")

	log.WithField("Code", astMod.String()).Debug("Server service handlers before template")

	// Index the handler functions, apply handler template for all function in service definition that are not defined in handler
	currentFuncs := astMod.IndexFunctions()
	code := astMod.Buffer()

	err = g.applyTemplateForMissingMeths(svcMethodsTemplPath, currentFuncs, code)
	if err != nil {
		return nil, errors.Wrap(err, "unable to apply service methods template")
	}

	// Insert updated Service interface
	outBuf, err := g.applyTemplate(svcInterfaceTemplPath, g.templateExec)
	if err != nil {
		return nil, errors.Wrap(err, "unable to apply service interface template")
	}

	code.Write(outBuf.Bytes())

	return code, nil
}

// updateClientMethods will update the functions within an existing
// client_handler.go file so that it contains exactly the fucntions passed in
// svcFuncs, no more, no less.
func (g *generator) updateClientMethods(clientHandler *string, svcFuncs []string) (outCode *bytes.Buffer, err error) {
	const clientMethodsTemplPath = "service/partial_template/client_handler.methods"
	astMod := astmodifier.New(clientHandler)

	// Remove functions no longer in definition
	astMod.RemoveFunctionsExecpt(svcFuncs)

	log.WithField("Code", astMod.String()).Debug("Client handlers before template")

	// Index handler functions, apply handler template for all function in
	// service definition that are not defined in handler
	currentFuncs := astMod.IndexFunctions()
	code := astMod.Buffer()

	err = g.applyTemplateForMissingMeths(clientMethodsTemplPath, currentFuncs, code)

	if err != nil {
		return nil, errors.Wrap(err, "unable to apply client methods template")
	}

	return code, nil
}

// GenerateResponseFiles applies all template files for the generated
// microservice and returns a slice containing each templated file as a
// CodeGeneratorResponse_File.
func (g *generator) GenerateResponseFiles() ([]truss.SimpleFile, error) {
	const serviceHandlerFilePath = "service/server/service.go"
	const clientHandlerFilePath = "service/client/client_handler.go"

	var codeGenFiles []truss.SimpleFile

	// serviceFunctions is used later as the master list of all service methods
	// which should exist within the `server/service.go` and
	// `client/client_handler.go` files.
	var serviceFunctions []string
	for _, meth := range g.templateExec.Service.Methods {
		serviceFunctions = append(serviceFunctions, meth.GetName())
	}
	serviceFunctions = append(serviceFunctions, "NewBasicService")

	serviceHandlerFile := getFileContentByName(serviceHandlerFilePath, g.previousFiles)
	clientHandlerFile := getFileContentByName(clientHandlerFilePath, g.previousFiles)

	for _, templateFilePath := range g.templateFileNames() {
		if filepath.Ext(templateFilePath) != ".gotemplate" {
			log.WithField("Template file", templateFilePath).Debug("Skipping rendering non-buildable partial template")
			continue
		}

		var generatedFilePath string
		var generatedCode *bytes.Buffer
		var err error

		if templateFilePath == serviceHandlerFilePath+"template" && serviceHandlerFile != nil {
			// If there's an existing service file, update its contents

			generatedFilePath = serviceHandlerFilePath
			generatedCode, err = g.updateServiceMethods(serviceHandlerFile, serviceFunctions)

			if err != nil {
				return nil, errors.Wrap(err, "could not modifiy service handler file")
			}

		} else if templateFilePath == clientHandlerFilePath+"template" && clientHandlerFile != nil {
			// If there's an existing client_handler file, update its contents

			generatedFilePath = clientHandlerFilePath
			generatedCode, err = g.updateClientMethods(clientHandlerFile, serviceFunctions)

			if err != nil {
				return nil, errors.Wrap(err, "could not modifiy client handler file")
			}

		} else {
			generatedFilePath = templateFilePath
			// Change file path from .gotemplate to .go
			generatedFilePath = strings.TrimSuffix(generatedFilePath, "template")

			generatedCode, err = g.applyTemplate(templateFilePath, g.templateExec)
			if err != nil {
				return nil, errors.Wrap(err, "could not render template")
			}
		}

		// Turn code buffer into string and format it
		code := generatedCode.String()
		formattedCode := formatCode(code)

		resp := truss.SimpleFile{
			Name:    &generatedFilePath,
			Content: &formattedCode,
		}

		codeGenFiles = append(codeGenFiles, resp)
	}

	return codeGenFiles, nil
}

// applyTemplateForMissingMeths accepts a funcIndex which represents functions
// already present in a gofile, applyTemplateForMissingMeths compares this map
// to the functions defined in the service and renders the template with the
// path of templPath, and appends this to passed code
func (g *generator) applyTemplateForMissingMeths(templPath string, funcIndex map[string]bool, code *bytes.Buffer) error {
	var methodsToTemplate []*doctree.ServiceMethod
	for _, meth := range g.templateExec.Service.Methods {
		methName := meth.GetName()
		if funcIndex[methName] == false {
			methodsToTemplate = append(methodsToTemplate, meth)
			log.WithField("Method", methName).Info("Rendering template for method")
		} else {
			log.WithField("Method", methName).Info("Handler method already exists")
		}
	}

	// Create temporary templateExec with only the methods we want to append
	// We must also dereference the templateExec's Service and change our newly created
	// Service's pointer to it's messages to be methodsToTemplate
	templateExecWithOnlyMissingMethods := g.templateExec
	tempService := *g.templateExec.Service
	tempService.Methods = methodsToTemplate
	templateExecWithOnlyMissingMethods.Service = &tempService

	// Apply the template and write it to code
	templateOut, err := g.applyTemplate(templPath, templateExecWithOnlyMissingMethods)
	if err != nil {
		return errors.Wrapf(err, "could not apply template for missing methods: %v", templPath)
	}

	_, err = code.Write(templateOut.Bytes())
	if err != nil {
		return errors.Wrap(err, "could not append rendered template to code")
	}

	return nil
}

// applyTemplate accepts a path to a template and an interface to execute on that template
// returns a *bytes.Buffer containing the results of that execution
func (g *generator) applyTemplate(templateFilePath string, executor interface{}) (*bytes.Buffer, error) {

	templateBytes, err := g.templateFile(templateFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to find template file: %v", templateFilePath)
	}

	templateString := string(templateBytes)
	codeTemplate := template.Must(template.New(templateFilePath).Funcs(g.templateFuncMap).Parse(templateString))

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
func formatCode(code string) string {
	formatted, err := format.Source([]byte(code))

	if err != nil {
		log.WithError(err).Warn("Code formatting error, generated service will not build, outputting unformatted code")
		// Set formatted to code so at least we get something to examine
		formatted = []byte(code)
	}

	return string(formatted)
}
