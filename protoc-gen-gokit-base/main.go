package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gengo/grpc-gateway/protoc-gen-grpc-gateway/descriptor"
	_ "github.com/gengo/grpc-gateway/protoc-gen-grpc-gateway/generator"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	_ "github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

var (
	errNoTargetService = errors.New("no target service defined in the file")
)

// parseReq reads io.Reader r into memory and attempts to marshal
// that input into a protobuf plugin CodeGeneratorRequest
func parseReq(r io.Reader) (*plugin.CodeGeneratorRequest, error) {
	glog.V(1).Info("Parsing code generator request")
	input, err := ioutil.ReadAll(r)
	if err != nil {
		glog.Errorf("Failed to read code generator request: %v", err)
		return nil, err
	}
	req := new(plugin.CodeGeneratorRequest)
	if err = proto.Unmarshal(input, req); err != nil {
		glog.Errorf("Failed to unmarshal code generator request: %v", err)
		return nil, err
	}
	glog.V(1).Info("Parsed code generator request")
	return req, nil
}

var (
	response = string("")
)

func logf(format string, args ...interface{}) {
	response += fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, format, args...)
}

var headerTemplate *template.Template

func main() {
	defer glog.Flush()
	glog.V(1).Info("Processing code generator request")

	registry := descriptor.NewRegistry()
	request, err := parseReq(os.Stdin)
	if err != nil {
		glog.Fatal(err)
	}

	g := New(registry)

	if err := registry.Load(request); err != nil {
		return
	}

	var targets []*descriptor.File
	for _, target := range request.FileToGenerate {
		f, err := registry.LookupFile(target)
		if err != nil {
			glog.Fatal(err)
		}
		targets = append(targets, f)
	}

	logf("%v\n", targets)
	logf("%v\n", len(targets))
	glog.V(1).Info("Building Output")

	var codeGenFiles []*plugin.CodeGeneratorResponse_File
	for _, file := range AssetNames() {
		//logf("%v\n", paths)
		curResponseFile := plugin.CodeGeneratorResponse_File{}

		// Remove "template/" so that generated files do not include that directory
		d := strings.TrimPrefix(file, "template/")
		curResponseFile.Name = &d

		bytesOfFile, _ := Asset(file)
		stringFile := string(bytesOfFile)
		if path.Base(file) == "main.go" {
			headerTemplate, _ = template.New("main.go").Parse(stringFile)
			stringFile, _ = g.MyGenerate(targets)
		}
		curResponseFile.Content = &stringFile

		codeGenFiles = append(codeGenFiles, &curResponseFile)
	}

	output := &plugin.CodeGeneratorResponse{
		File: codeGenFiles,
	}

	buf, err := proto.Marshal(output)
	if err != nil {
		glog.Fatal(err)
	}

	if _, err := os.Stdout.Write(buf); err != nil {
		glog.Fatal(err)
	}
}

type generator struct {
	reg         *descriptor.Registry
	baseImports []descriptor.GoPackage
}

// New returns a new generator which generates grpc gateway files.
func New(reg *descriptor.Registry) *generator {
	var imports []descriptor.GoPackage
	for _, pkgpath := range []string{
		//"io",
		//"net/http",
		//"github.com/gengo/grpc-gateway/runtime",
		//"github.com/gengo/grpc-gateway/utilities",
		//"github.com/golang/protobuf/proto",
		//"golang.org/x/net/context",
		//"google.golang.org/grpc",
		//"google.golang.org/grpc/codes",
		//"google.golang.org/grpc/grpclog",
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
	return &generator{reg: reg, baseImports: imports}
}

func (g *generator) MyGenerate(targets []*descriptor.File) (string, error) {
	//var files []*plugin.CodeGeneratorResponse_File
	for _, file := range targets {
		glog.V(1).Infof("Processing %s", file.GetName())
		code, err := g.generate(file)
		if err == errNoTargetService {
			glog.V(1).Infof("%s: %v", file.GetName(), err)
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

// Move all generation to this function
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
	logf("%v\n", p)
	if err := headerTemplate.Execute(w, p); err != nil {
		return "", err
	}
	logf("%v\n", w.String())
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
