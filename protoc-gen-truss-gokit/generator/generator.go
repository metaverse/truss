package generator

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/TuneLab/gob/protoc-gen-truss-gokit/astmodifier"
	templateFileAssets "github.com/TuneLab/gob/protoc-gen-truss-gokit/template"

	"github.com/TuneLab/gob/gendoc/doctree"
	"github.com/TuneLab/gob/protoc-gen-truss-gokit/generator/clientarggen"
	generatego "github.com/golang/protobuf/protoc-gen-go/generator"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"

	log "github.com/Sirupsen/logrus"
)

func init() {
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stderr)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})
}

type generator struct {
	files             []*doctree.ProtoFile
	outputDirName     string
	templateFileNames func() []string
	templateFile      func(string) ([]byte, error)
	templateFuncMap   template.FuncMap
	templateExec      templateExecutor
}

// templateExecutor is passed to templates as the executing struct
// Its fields and methods are used to modify the template
type templateExecutor struct {
	// Import path for handler package
	HandlerImport string
	// Import path for generated packages
	GeneratedImport string
	// GRPC/Protobuff service, with all parameters and return values accessible
	Service    *doctree.ProtoService
	ClientArgs *clientarggen.ClientServiceArgs
}

// New returns a new generator which generates grpc gateway files.
func New(files []*doctree.ProtoFile, outputDirName string) *generator {
	var service *doctree.ProtoService
	log.WithField("File Count", len(files)).Info("Files are being processed")

	// Find the service to be attached to the templateExecutor
	for _, file := range files {

		log.WithFields(log.Fields{
			"File":          file.GetName(),
			"Service Count": len(file.Services),
		}).Info("File being processed")

		if len(file.Services) > 0 {
			service = file.Services[0]
			log.WithField("Service", service.GetName()).Info("Service Discoved")
		}
	}

	if service == nil {
		log.Fatal("No service discovered, aborting...")
	}

	// Get the working directory and the go import paths for the working directory
	// e.g.
	// Working directory: (wd)
	// /home/adamryman/projects/go/src/github.com/TuneLab/gob/protoc-gen-truss-gokit/
	// $GOPATH: (goPath)
	// /home/adamryman/projects/go/
	// So strings.TrimPrefix(wd, goPath+"/src/") = github.com/TuneLab/gob/protoc-gen-truss-gokit
	wd, _ := os.Getwd()
	goPath := os.Getenv("GOPATH")
	wdImportString := strings.TrimPrefix(wd, goPath+"/src/")

	// Then add onto the wdImportString the outputDirName to get our baseImportString
	baseImportString := wdImportString + "/" + outputDirName

	log.WithField("Output dir", outputDirName).Info("Output directory")
	log.WithField("serviceImportPath", baseImportString).Info("Service path")

	// import path for generated code with handlers that the user can edit
	handlerImportString := baseImportString
	// import path for generated code that user should not edit
	generatedImportString := baseImportString + "/DONOTEDIT"

	funcMap := template.FuncMap{
		"ToLower":    strings.ToLower,
		"Title":      strings.Title,
		"GoName":     generatego.CamelCase,
		"TrimPrefix": strings.TrimPrefix,
	}

	return &generator{
		files:             files,
		outputDirName:     outputDirName,
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

// fileExists checks if a file at the given path exists. Returns true if the
// file exists, and false if the file does not exist.
func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

// updateServiceMethods will update the functions within an existing service.go
// file so it contains all the functions within svcFuncs and ONLY those
// functions within svcFuncs
func (g *generator) updateServiceMethods(svcPath string, svcFuncs []string) (outPath string, outCode string) {
	log.Info("server/service.go exists")
	astMod := astmodifier.New(svcPath)

	// Remove functions no longer in definition and remove Service interface
	astMod.RemoveFunctionsExecpt(svcFuncs)
	astMod.RemoveInterface("Service")

	log.WithField("Code", astMod.String()).Debug("Server service handlers before template")

	// Index the handler functions, apply handler template for all function in service definition that are not defined in handler
	currentFuncs := astMod.IndexFunctions()
	code := astMod.Buffer()
	code = g.applyTemplateForMissingServiceMethods("template_files/partial_template/service.methods", currentFuncs, code)

	// Insert updated Service interface
	outBuf := g.applyTemplate("template_files/partial_template/service.interface", g.templateExec)
	code.WriteString(outBuf)

	// Get file ready to write
	outPath = "server/service.go"
	outCode = formatCode(code.String())

	return outPath, outCode
}

// updateClientMethods will update the functions within an existing
// client_handler.go file so that it contains exactly the fucntions passed in
// svcFuncs, no more, no less.
func (g *generator) updateClientMethods(clientPath string, svcFuncs []string) (outPath string, outCode string) {
	log.Info("client/client_handler.go exists")
	astMod := astmodifier.New(clientPath)

	// Remove functions no longer in definition
	astMod.RemoveFunctionsExecpt(svcFuncs)

	log.WithField("Code", astMod.String()).Debug("Client handlers before template")

	// Index handler functions, apply handler template for all function in
	// service definition that are not defined in handler
	currentFuncs := astMod.IndexFunctions()
	code := astMod.Buffer()
	code = g.applyTemplateForMissingServiceMethods("template_files/partial_template/client_handler.methods", currentFuncs, code)

	// Get file ready to write
	outPath = "client/client_handler.go"
	outCode = formatCode(code.String())
	return outPath, outCode
}

// GenerateResponseFiles applies all template files for the generated
// microservice and returns a slice containing each templated file as a
// CodeGeneratorResponse_File.
func (g *generator) GenerateResponseFiles() ([]*plugin.CodeGeneratorResponse_File, error) {
	var codeGenFiles []*plugin.CodeGeneratorResponse_File

	wd, _ := os.Getwd()

	clientPath := wd + "/" + g.outputDirName + "/client/client_handler.go"
	servicePath := wd + "/" + g.outputDirName + "/server/service.go"

	// serviceFunctions is used later as the master list of all service methods
	// which should exist within the `server/service.go` and
	// `client/client_handler.go` files.
	var serviceFunctions []string
	for _, meth := range g.templateExec.Service.Methods {
		serviceFunctions = append(serviceFunctions, meth.GetName())
	}
	serviceFunctions = append(serviceFunctions, "NewBasicService")

	for _, templateFilePath := range g.templateFileNames() {
		if filepath.Ext(templateFilePath) != ".gotemplate" {
			log.WithField("Template file", templateFilePath).Debug("Skipping rendering non-buildable partial template")
			continue
		}

		var generatedFilePath string
		var generatedCode string

		if filepath.Base(templateFilePath) == "service.gotemplate" && fileExists(servicePath) {
			// If there's an existing service file, update its contents
			generatedFilePath, generatedCode = g.updateServiceMethods(servicePath, serviceFunctions)

		} else if filepath.Base(templateFilePath) == "client_handler.gotemplate" && fileExists(clientPath) {
			// If there's an existing client_handler file, update its contents
			generatedFilePath, generatedCode = g.updateClientMethods(clientPath, serviceFunctions)

		} else {
			// Remove "template_files/" so that generated files do not include that directory
			generatedFilePath = strings.TrimPrefix(templateFilePath, "template_files/")
			// Change file path from .gotemplate to .go
			generatedFilePath = strings.TrimSuffix(generatedFilePath, "template")

			generatedCode = g.applyTemplate(templateFilePath, g.templateExec)
			generatedCode = formatCode(generatedCode)
		}

		generatedFilePath = g.outputDirName + "/" + generatedFilePath
		resp := plugin.CodeGeneratorResponse_File{
			Name:    &generatedFilePath,
			Content: &generatedCode,
		}
		codeGenFiles = append(codeGenFiles, &resp)
	}

	return codeGenFiles, nil
}

func (g *generator) applyTemplateForMissingServiceMethods(templateFilePath string, functionIndex map[string]bool, code *bytes.Buffer) *bytes.Buffer {
	var methodsToTemplate []*doctree.ServiceMethod
	for _, meth := range g.templateExec.Service.Methods {
		methName := meth.GetName()
		if functionIndex[methName] == false {
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
	templateOut := g.applyTemplate(templateFilePath, templateExecWithOnlyMissingMethods)
	code.WriteString(templateOut)
	return code
}

func (g *generator) applyTemplate(templateFilePath string, executor interface{}) string {
	templateBytes, _ := g.templateFile(templateFilePath)
	templateString := string(templateBytes)
	codeTemplate := template.Must(template.New(templateFilePath).Funcs(g.templateFuncMap).Parse(templateString))

	outputBuffer := bytes.NewBuffer(nil)
	err := codeTemplate.Execute(outputBuffer, executor)
	if err != nil {
		log.WithError(err).Fatal("Template Error")
	}

	return outputBuffer.String()
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
