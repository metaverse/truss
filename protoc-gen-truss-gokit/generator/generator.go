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
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"

	log "github.com/Sirupsen/logrus"
)

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stderr)
}

type generator struct {
	files             []*doctree.ProtoFile
	outputDirName     string
	templateFileNames func() []string
	templateFile      func(string) ([]byte, error)
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
	Service *doctree.ProtoService
	// Contains the strings.ToLower() method for lowercasing Service names, methods, and fields
	Strings stringsTemplateMethods
}

// Purely a wrapper for the strings.ToLower() method
type stringsTemplateMethods struct {
	ToLower func(string) string
}

// New returns a new generator which generates grpc gateway files.
func New(files []*doctree.ProtoFile, outputDirName string) *generator {
	var service *doctree.ProtoService
	log.WithField("File Count", len(files)).Info("Files are being processed")

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

	wd, _ := os.Getwd()
	goPath := os.Getenv("GOPATH")
	baseImportPath := strings.TrimPrefix(wd, goPath+"/src/")
	serviceImportPath := baseImportPath + "/" + outputDirName
	log.WithField("Output dir", outputDirName).Info("Output directory")
	log.WithField("serviceImportPath", serviceImportPath).Info("Service path")
	// import path for generated code that the user can edit
	handlerImportPath := serviceImportPath
	// import path for generated code that user should not edit
	generatedImportPath := serviceImportPath + "/DONOTEDIT"

	// Attaching the strings.ToLower method so that it can be used in templae execution
	stringsMethods := stringsTemplateMethods{
		ToLower: strings.ToLower,
	}

	return &generator{
		files:             files,
		outputDirName:     outputDirName,
		templateFileNames: templateFileAssets.AssetNames,
		templateFile:      templateFileAssets.Asset,
		templateExec: templateExecutor{
			HandlerImport:   handlerImportPath,
			GeneratedImport: generatedImportPath,
			Service:         service,
			Strings:         stringsMethods,
		},
	}
}

func (g *generator) GenerateResponseFiles() ([]*plugin.CodeGeneratorResponse_File, error) {
	var codeGenFiles []*plugin.CodeGeneratorResponse_File

	wd, _ := os.Getwd()

	var clientHandlerExists bool
	clientPath := wd + "/" + g.outputDirName + "/client/client_handler.go"
	if _, err := os.Stat(clientPath); err == nil {
		clientHandlerExists = true
	}

	var serviceHandlerExists bool
	servicePath := wd + "/" + g.outputDirName + "/server/service.go"
	if _, err := os.Stat(servicePath); err == nil {
		serviceHandlerExists = true
	}

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

		if filepath.Base(templateFilePath) == "service.gotemplate" && serviceHandlerExists {
			log.Info("server/service.go exists")
			astMod := astmodifier.New(servicePath)

			// Remove functions no longer in definition and remove Service interface
			astMod.RemoveFunctionsExecpt(serviceFunctions)
			astMod.RemoveInterface("Service")

			log.WithField("Code", astMod.String()).Debug("Server service handlers before template")

			// Index handler functions, apply handler template for all function in service definition that are not defined in handler
			currentFuncs := astMod.IndexFunctions()
			code := astMod.Buffer()
			code = g.applyTemplateForMissingServiceMethods("template_files/partial_template/service.method", currentFuncs, code)

			// Insert updated Service interface
			templateOut := g.applyTemplate("template_files/partial_template/service.interface", g.templateExec)
			code.WriteString(templateOut)

			// Get file ready to write
			generatedFilePath = "server/service.go"
			generatedCode = formatCode(code.String())

		} else if filepath.Base(templateFilePath) == "client_handler.gotemplate" && clientHandlerExists {
			log.Info("client/client_handler.go exists")
			astMod := astmodifier.New(clientPath)

			// Remove functions no longer in definition
			astMod.RemoveFunctionsExecpt(serviceFunctions)

			log.WithField("Code", astMod.String()).Debug("Client handlers before template")

			// Index handler functions, apply handler template for all function in service definition that are not defined in handler
			currentFuncs := astMod.IndexFunctions()
			code := astMod.Buffer()
			code = g.applyTemplateForMissingServiceMethods("template_files/partial_template/client_handler.method", currentFuncs, code)

			// Get file ready to write
			generatedFilePath = "client/client_handler.go"
			generatedCode = formatCode(code.String())

		} else {
			// Remove "template_files/" so that generated files do not include that directory
			generatedFilePath = strings.TrimPrefix(templateFilePath, "template_files/")

			// Change file path from .gotemplate to .go
			generatedFilePath = strings.TrimSuffix(generatedFilePath, "template")

			generatedCode = g.applyTemplate(templateFilePath, g.templateExec)

			generatedCode = formatCode(generatedCode)
		}
		generatedFilePath = g.outputDirName + "/" + generatedFilePath

		curResponseFile := plugin.CodeGeneratorResponse_File{
			Name:    &generatedFilePath,
			Content: &generatedCode,
		}

		codeGenFiles = append(codeGenFiles, &curResponseFile)
	}

	return codeGenFiles, nil
}

func (g *generator) applyTemplateForMissingServiceMethods(templateFilePath string, functionIndex map[string]bool, code *bytes.Buffer) *bytes.Buffer {
	for _, meth := range g.templateExec.Service.Methods {
		methName := meth.GetName()
		if functionIndex[methName] == false {
			log.WithField("Method", methName).Info("Rendering template for method")
			templateOut := g.applyTemplate(templateFilePath, meth)
			code.WriteString(templateOut)
		} else {
			log.WithField("Method", methName).Info("Handler method already exists")
		}
	}
	return code
}

func (g *generator) applyTemplate(templateFilePath string, executor interface{}) string {

	templateBytes, _ := g.templateFile(templateFilePath)

	templateString := string(templateBytes)

	codeTemplate := template.Must(template.New(templateFilePath).Parse(templateString))

	outputBuffer := bytes.NewBuffer(nil)
	err := codeTemplate.Execute(outputBuffer, executor)
	if err != nil {
		log.WithError(err).Fatal("Template Error")
	}

	return outputBuffer.String()
}

func formatCode(code string) string {
	formatted, err := format.Source([]byte(code))

	if err != nil {
		log.WithError(err).Warn("Code formatting error, generated service will not build, outputting unformatted code")
		// Set formatted to code so at least we get something to examine
		formatted = []byte(code)
	}

	return string(formatted)
}
