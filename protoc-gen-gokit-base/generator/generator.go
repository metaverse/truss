package generator

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"strings"
	"text/template"

	templateFileAssets "github.com/TuneLab/gob/protoc-gen-gokit-base/template"
	"github.com/gengo/grpc-gateway/protoc-gen-grpc-gateway/descriptor"
	"github.com/golang/glog"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

var (
	headerTemplate     *template.Template
	errNoTargetService = errors.New("no target service defined in the file")
)

type generator struct {
	reg               *descriptor.Registry
	baseImports       []descriptor.GoPackage
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

var (
	response = string("")
)

// Leland Batey's log to os.Stderr
func logf(format string, args ...interface{}) {
	response += fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, format, args...)
}

// New returns a new generator which generates grpc gateway files.
func New(reg *descriptor.Registry) *generator {
	var imports []descriptor.GoPackage
	return &generator{
		reg:               reg,
		baseImports:       imports,
		templateFileNames: templateFileAssets.AssetNames,
		templateFile:      templateFileAssets.Asset,
		templateExec:      templateExecutor{},
	}
}

func (g *generator) GenerateResponseFiles(targets []*descriptor.File) ([]*plugin.CodeGeneratorResponse_File, error) {
	var codeGenFiles []*plugin.CodeGeneratorResponse_File

	for _, file := range g.templateFileNames() {
		logf("%v\n", file)
		curResponseFile := plugin.CodeGeneratorResponse_File{}

		// Remove "template_files/" so that generated files do not include that directory
		d := strings.TrimPrefix(file, "template_files/")
		curResponseFile.Name = &d

		// Get the bytes from the file we are working on
		// then turn it into a string to build a template out of it
		bytesOfFile, _ := g.templateFile(file)
		stringFile := string(bytesOfFile)

		// Currently only templating main.go
		stringFile, _ = myGenerate(targets, file, bytesOfFile)
		curResponseFile.Content = &stringFile

		codeGenFiles = append(codeGenFiles, &curResponseFile)
	}

	return codeGenFiles, nil
}

func myGenerate(targets []*descriptor.File, templateName string, templateBytes []byte) (string, error) {

	templateString := string(templateBytes)

	headerTemplate = template.Must(template.New(templateName).Parse(templateString))

	for _, file := range targets {
		glog.V(1).Infof("Processing %s", file.GetName())
		code, err := applyTemplate(file)
		if err == errNoTargetService {
			continue
		}
		if err != nil {
			return "", err
		}
		formatted, err := format.Source([]byte(code))
		// MY RETURN SHORT CIRCUT
		return string(formatted), err
	}
	return "", nil
}

func applyTemplate(file *descriptor.File) (string, error) {
	w := bytes.NewBuffer(nil)
	if err := headerTemplate.Execute(w, templateExecutor{}); err != nil {
		return "FAIL", err
	}
	return w.String(), nil
}

type binding struct {
	*descriptor.Binding
}
