package generator

import (
	"bytes"
	"go/format"
	"os"
	"strings"
	"text/template"

	templateFileAssets "github.com/TuneLab/gob/protoc-gen-gokit-base/template"
	"github.com/TuneLab/gob/protoc-gen-gokit-base/util"

	"github.com/gengo/grpc-gateway/protoc-gen-grpc-gateway/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

type generator struct {
	reg               *descriptor.Registry
	files             []*descriptor.File
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
	Service *descriptor.Service
	// Contains the strings.ToLower() method for lowercasing Service names, methods, and fields
	Strings stringsTemplateMethods
}

// Purely a wrapper for the strings.ToLower() method
type stringsTemplateMethods struct {
	ToLower func(string) string
}

// Get working directory, trim off GOPATH, add generate.
// This should be the absolute path for the relative package dependencies

// New returns a new generator which generates grpc gateway files.
func New(reg *descriptor.Registry, files []*descriptor.File) *generator {
	var service *descriptor.Service
	util.Logf("There are %v file(s) being processed\n", len(files))
	for _, file := range files {
		util.Logf("File: %v\n", file.GetName())
		util.Logf("This file has %v service(s)\n", len(file.Services))
		if len(file.Services) > 0 {
			service = file.Services[0]
			util.Logf("\tNamed: %v\n", service.GetName())
		}
	}

	wd, _ := os.Getwd()
	goPath := os.Getenv("GOPATH")
	baseImportPath := strings.TrimPrefix(wd, goPath+"/src/")
	handlerImportPath := baseImportPath
	generatedImportPath := baseImportPath + "/DONOTEDIT"

	// Attaching the strings.ToLower method so that it can be used in template execution
	stringsMethods := stringsTemplateMethods{
		ToLower: strings.ToLower,
	}

	return &generator{
		reg:               reg,
		files:             files,
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

func (g *generator) GenerateResponseFiles(targets []*descriptor.File) ([]*plugin.CodeGeneratorResponse_File, error) {
	var codeGenFiles []*plugin.CodeGeneratorResponse_File

	g.printAllServices()

	for _, templateFile := range g.templateFileNames() {
		curResponseFile := plugin.CodeGeneratorResponse_File{}

		// Remove "template_files/" so that generated files do not include that directory
		d := strings.TrimPrefix(templateFile, "template_files/")
		curResponseFile.Name = &d

		// Get the bytes from the file we are working on
		// then turn it into a string to build a template out of it
		bytesOfFile, _ := g.templateFile(templateFile)
		stringFile := string(bytesOfFile)

		// Currently only templating main.go
		stringFile, _ = g.generate(templateFile, bytesOfFile)
		curResponseFile.Content = &stringFile

		codeGenFiles = append(codeGenFiles, &curResponseFile)
	}

	return codeGenFiles, nil
}

func (g *generator) printAllServices() {
	if g.templateExec.Service != nil {
		util.Logf("\tService: %v\n", g.templateExec.Service.GetName())
		for _, method := range g.templateExec.Service.Methods {
			util.Logf("\t\tMethod: %v\n", method.GetName())
			util.Logf("\t\t\t Request: %v\n", method.RequestType.GetName())
			util.Logf("\t\t\t Response: %v\n", method.ResponseType.GetName())
			if method.Options != nil {
				util.Logf("\t\t\t\tOptions: %v\n", method.Options.String())
			}

		}
	}
}

func (g *generator) generate(templateName string, templateBytes []byte) (string, error) {

	templateString := string(templateBytes)

	codeTemplate := template.Must(template.New(templateName).Parse(templateString))

	w := bytes.NewBuffer(nil)
	err := codeTemplate.Execute(w, g.templateExec)
	if err != nil {
		util.Logf("\nTEMPLATE ERROR\n", "")
		util.Logf("\n%v\n", templateName)
		util.Logf("\n%v\n\n", err.Error())
		panic(err)
		return "", err
	}

	code := w.String()

	formatted, err := format.Source([]byte(code))

	if err != nil {
		util.Logf("\nCODE FORMATTING ERROR\n", "")
		util.Logf("\n%v\n", code)
		util.Logf("\n%v\n", err.Error())
		// Set formatted to code so at least we get something to examine
		formatted = []byte(code)
	}

	return string(formatted), err
}
