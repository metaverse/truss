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
}

// Get working directory, trim off GOPATH, add generate.
// This should be the absolute path for the relative package dependencies
func (t templateExecutor) AbsoluteRelativeImportPath() string {
	wd, _ := os.Getwd()
	goPath := os.Getenv("GOPATH")
	importPath := strings.TrimPrefix(wd, goPath+"/src/")
	importPath = importPath + "/generate/"

	return importPath
}

// New returns a new generator which generates grpc gateway files.
func New(reg *descriptor.Registry, files []*descriptor.File) *generator {
	return &generator{
		reg:               reg,
		files:             files,
		templateFileNames: templateFileAssets.AssetNames,
		templateFile:      templateFileAssets.Asset,
		templateExec:      templateExecutor{},
	}
}

func (g *generator) GenerateResponseFiles(targets []*descriptor.File) ([]*plugin.CodeGeneratorResponse_File, error) {
	var codeGenFiles []*plugin.CodeGeneratorResponse_File

	g.printAllServices()

	for _, file := range g.templateFileNames() {
		util.Logf("%v\n", file)
		curResponseFile := plugin.CodeGeneratorResponse_File{}

		// Remove "template_files/" so that generated files do not include that directory
		d := strings.TrimPrefix(file, "template_files/")
		curResponseFile.Name = &d

		// Get the bytes from the file we are working on
		// then turn it into a string to build a template out of it
		bytesOfFile, _ := g.templateFile(file)
		stringFile := string(bytesOfFile)

		// Currently only templating main.go
		stringFile, _ = generate(targets, file, bytesOfFile)
		curResponseFile.Content = &stringFile

		codeGenFiles = append(codeGenFiles, &curResponseFile)
	}

	return codeGenFiles, nil
}

func (g *generator) printAllServices() {
	for _, file := range g.files {
		util.Logf("File: %v\n", file.GetName())
		for _, service := range file.Services {
			util.Logf("\tService: %v\n", service.GetName())
			for _, method := range service.Methods {
				util.Logf("\t\tMethod: %v\n", method.GetName())
				util.Logf("\t\t\t Request: %v\n", method.RequestType.GetName())
				util.Logf("\t\t\t Response: %v\n", method.ResponseType.GetName())
				util.Logf("\t\t\t\tOptions: %v\n", method.Options.String())
			}
		}
	}
}

func generate(targets []*descriptor.File, templateName string, templateBytes []byte) (string, error) {

	templateString := string(templateBytes)

	codeTemplate := template.Must(template.New(templateName).Parse(templateString))

	w := bytes.NewBuffer(nil)
	err := codeTemplate.Execute(w, templateExecutor{})
	if err != nil {
		return "", err
	}
	code := w.String()
	formatted, err := format.Source([]byte(code))

	return string(formatted), err
}
