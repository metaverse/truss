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

type templateExecutor struct {
	AbsoluteRelativeImportPath string
	Service                    *descriptor.Service
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
	importPath := strings.TrimPrefix(wd, goPath+"/src/")
	importPath = importPath + "/generate"

	return &generator{
		reg:               reg,
		files:             files,
		templateFileNames: templateFileAssets.AssetNames,
		templateFile:      templateFileAssets.Asset,
		templateExec: templateExecutor{
			AbsoluteRelativeImportPath: importPath,
			Service:                    service,
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
		util.Logf("\n%v\n", templateName)
		util.Logf("\n%v\n\n", err.Error())
		panic(err)
		return "", err
	}

	code := w.String()

	formatted, err := format.Source([]byte(code))

	return string(formatted), err
}
