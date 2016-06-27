package generator

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	templateFileAssets "github.com/TuneLab/gob/protoc-gen-gokit-base/template"
	"github.com/gengo/grpc-gateway/protoc-gen-grpc-gateway/descriptor"
	"github.com/gogo/protobuf/proto"
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
	// Loop through base golang imports and add them to the generator
	// If there are conflicts, use Alias function of registry
	for _, pkgpath := range []string{
		"fmt",
		"log",
		"math/rand",
		"net",
		"os",
		"os/signal",
		"strconv",
		"syscall",
		"time",

		"github.com/go-kit/kit/log",
		"github.com/go-kit/kit/log/levels",
		"github.com/TuneLab/gob/protoc-gen-gokit-base/generate/controller",
		"github.com/TuneLab/gob/protoc-gen-gokit-base/generate/pb",
		"github.com/TuneLab/gob/protoc-gen-gokit-base/generate/server",
		"google.golang.org/grpc",
	} {
		pkg := descriptor.GoPackage{
			Path: pkgpath,
			Name: path.Base(pkgpath),
		}
		if err := reg.ReserveGoPackageAlias(pkg.Name, pkg.Path); err != nil {
			for i := 0; ; i++ {
				alias := fmt.Sprintf("%s_%d", pkg.Name, i)
				if err := reg.ReserveGoPackageAlias(alias, pkg.Path); err != nil {
					continue
				}
				pkg.Alias = alias
				break
			}
		}
		imports = append(imports, pkg)
	}
	return &generator{
		reg:               reg,
		baseImports:       imports,
		templateFileNames: templateFileAssets.AssetNames,
		templateFile:      templateFileAssets.Asset,
	}
}

func (g *generator) GenerateResponseFiles(targets []*descriptor.File) ([]*plugin.CodeGeneratorResponse_File, error) {
	var codeGenFiles []*plugin.CodeGeneratorResponse_File
	for _, file := range g.templateFileNames() {
		//logf("%v\n", paths)
		curResponseFile := plugin.CodeGeneratorResponse_File{}

		// Remove "template_files/" so that generated files do not include that directory
		d := strings.TrimPrefix(file, "template_files/")
		curResponseFile.Name = &d

		// Get the bytes from the file we are working on
		// then turn it into a string to build a template out of it
		bytesOfFile, _ := g.templateFile(file)
		stringFile := string(bytesOfFile)

		// Currently only templating main.go
		if path.Base(file) == "main.go" {
			stringFile, _ = g.MyGenerate(targets, file, bytesOfFile)
		}
		curResponseFile.Content = &stringFile

		codeGenFiles = append(codeGenFiles, &curResponseFile)
	}

	return codeGenFiles, nil
}

func (g *generator) MyGenerate(targets []*descriptor.File, templateName string, templateBytes []byte) (string, error) {
	templateString := string(templateBytes)
	headerTemplate = template.Must(template.New(templateName).Parse(templateString))
	//var files []*plugin.CodeGeneratorResponse_File

	//logf("%v\n", templateString)
	for _, file := range targets {
		glog.V(1).Infof("Processing %s", file.GetName())
		code, err := g.generate(file)
		//logf("%v\n", code)
		if err == errNoTargetService {
			glog.V(1).Infof("%s: %v", file.GetName(), err)
			continue
		}
		if err != nil {
			return "", err
		}
		formatted, err := format.Source([]byte(code))
		//logf("%v\n", formatted)
		// MY RETURN SHORT CIRCUT
		return string(formatted), err
	}
	return "", nil
}

func (g *generator) generate(file *descriptor.File) (string, error) {
	pkgSeen := make(map[string]bool)
	var imports []descriptor.GoPackage
	for _, pkg := range g.baseImports {
		pkgSeen[pkg.Path] = true
		imports = append(imports, pkg)
	}
	for _, svc := range file.Services {
		for _, m := range svc.Methods {
			pkg := m.RequestType.File.GoPkg
			if pkg == file.GoPkg {
				continue
			}
			if pkgSeen[pkg.Path] {
				continue
			}
			pkgSeen[pkg.Path] = true
			imports = append(imports, pkg)
		}
	}
	return applyTemplate(param{File: file, Imports: imports})
}

type param struct {
	*descriptor.File
	Imports []descriptor.GoPackage
}

func applyTemplate(p param) (string, error) {
	w := bytes.NewBuffer(nil)
	//logf("%v\n", p.GetSourceCodeInfo())
	//logf("%v\n", p)
	if err := headerTemplate.Execute(w, p); err != nil {
		return "", err
	}
	//logf("%v\n", w.String())
	//var methodSeen bool
	//for _, svc := range p.Services {
	//for _, meth := range svc.Methods {
	//glog.V(2).Infof("Processing %s.%s", svc.GetName(), meth.GetName())
	//methodSeen = true
	//for _, b := range meth.Bindings {
	//if err := handlerTemplate.Execute(w, binding{Binding: b}); err != nil {
	//return "", err
	//}
	//}
	//}
	//}
	//if !methodSeen {
	//return "", errNoTargetService
	//}
	//if err := trailerTemplate.Execute(w, p.Services); err != nil {
	//return "", err
	//}
	return w.String(), nil
}

type binding struct {
	*descriptor.Binding
}

func (g *generator) Generate(targets []*descriptor.File) ([]*plugin.CodeGeneratorResponse_File, error) {
	var files []*plugin.CodeGeneratorResponse_File
	for _, file := range targets {
		glog.V(1).Infof("Processing %s", file.GetName())
		code, err := g.generate(file)
		if err == errNoTargetService {
			glog.V(1).Infof("%s: %v", file.GetName(), err)
			continue
		}
		if err != nil {
			return nil, err
		}
		formatted, err := format.Source([]byte(code))
		if err != nil {
			glog.Errorf("%v: %s", err, code)
			return nil, err
		}
		name := file.GetName()
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		output := fmt.Sprintf("%s.pb.gw.go", base)
		files = append(files, &plugin.CodeGeneratorResponse_File{
			Name:    proto.String(output),
			Content: proto.String(string(formatted)),
		})
		glog.V(1).Infof("Will emit %s", output)
	}
	return files, nil
}
