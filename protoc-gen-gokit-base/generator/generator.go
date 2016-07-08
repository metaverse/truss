package generator

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	templateFileAssets "github.com/TuneLab/gob/protoc-gen-gokit-base/template"
	"github.com/TuneLab/gob/protoc-gen-gokit-base/util"

	"github.com/gengo/grpc-gateway/protoc-gen-grpc-gateway/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"

	_ "github.com/davecgh/go-spew/spew"
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
	// import path for generated code that the user can edit
	handlerImportPath := baseImportPath
	// import path for generated code that user should not edit
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

	wd, _ := os.Getwd()
	servicePath := wd + "/service.go"
	for _, templateFile := range g.templateFileNames() {

		// If service.go does not exist, generate all files
		// If template file is not service.go then generate the file
		// If service.go exists and the template file is service.go then skip
		if _, err := os.Stat(servicePath); os.IsNotExist(err) || filepath.Base(templateFile) != "service.go" {
			if filepath.Ext(templateFile) != ".go" {
				util.Logf("%v: is not a go file, skipping...\n", templateFile)
				continue
			}

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
		} else {

			util.Log("-------------------------------- service.go exists, not overwriting... ----------------------------------")
			// Steps that this block of code executes
			// 1. service.go is parsed into an Ast and stored in fileAst
			// 2. We create a map of methods in the protobuf file
			// 3. We create a walker which walks the ast, saving methods it finds, also deleting the service interface as it will be retemplated
			// 4. The ast is pretty printed into a buffer
			// 5. For every handler that is in the protobuf but not in service.go we template in a handler for that method
			// 6. The service interface template is added to the end
			// 7. The file is formatted

			fset := token.NewFileSet()
			fileAst, _ := parser.ParseFile(fset, servicePath, nil, 0)
			if err != nil {
				util.Log(err)
				panic(err)
			}

			protobufMethods := make(map[string]bool)
			for _, meth := range g.templateExec.Service.Methods {
				protobufMethods[meth.GetName()] = true
			}

			walker := &methodVisitor{
				handlerMethods:  make(map[string]bool),
				protobufMethods: protobufMethods,
			}

			serviceCode := bytes.NewBuffer(nil)
			//err = printer.Fprint(serviceCode, fset, fileAst)
			//if err != nil {
			//panic(err)
			//}

			//util.Logf("service code before walk:\n%v\n", serviceCode.String())

			ast.Walk(walker, fileAst)

			serviceCode = bytes.NewBuffer(nil)

			err = printer.Fprint(serviceCode, fset, fileAst)
			if err != nil {
				panic(err)
			}

			util.Logf("service code before temp:\n%v\n", serviceCode.String())
			for _, meth := range g.templateExec.Service.Methods {
				methName := meth.GetName()
				if walker.handlerMethods[methName] == false {
					templateOut := g.applyTemplate("template_files/handler.method", meth)
					serviceCode.Write(templateOut)
				} else {
					util.Logf("%v Exists\n", methName)
				}
			}

			templateOut := g.applyTemplate("template_files/service.interface", g.templateExec)
			serviceCode.Write(templateOut)

			formatted, err := format.Source(serviceCode.Bytes())

			if err != nil {
				util.Logf("\nCODE FORMATTING ERROR\n", "")
				util.Logf("\n%v\n", err.Error())
				// Set formatted to code so at least we get something to examine
				formatted = serviceCode.Bytes()
			}
			curResponseFile := plugin.CodeGeneratorResponse_File{}

			fileName := "service.go"
			curResponseFile.Name = &fileName

			stringFile := string(formatted)
			curResponseFile.Content = &stringFile

			codeGenFiles = append(codeGenFiles, &curResponseFile)
		}
	}

	return codeGenFiles, nil
}

func (g generator) applyTemplate(templateFile string, executor interface{}) []byte {
	templateBytes, _ := g.templateFile(templateFile)
	templateString := string(templateBytes)

	templ := template.Must(template.New(templateFile).Parse(templateString))
	outputBuffer := bytes.NewBuffer(nil)

	err := templ.Execute(outputBuffer, executor)
	if err != nil {
		util.Logf("\nTEMPLATE ERROR\n", "")
		util.Logf("\n%v\n", "ServiceRPC")
		util.Logf("\n%v\n\n", err.Error())
		panic(err)
	}

	return outputBuffer.Bytes()
}

type methodVisitor struct {
	handlerMethods  map[string]bool
	protobufMethods map[string]bool
	callNumber      int
}

func (v *methodVisitor) Visit(node ast.Node) ast.Visitor {
	v.callNumber = v.callNumber + 1
	serviceInterfaceFound := false
	var declareIndexToDelete int
	var funcsToDelete []int
	if file, ok := node.(*ast.File); ok {

		//fmt.Println(file.Pos())
		//fmt.Println(file.End())
		util.Log("---------- File ----------------")
		for i, decs := range file.Decls {
			//fmt.Printf("\t---------- Decls %v ---------------\n", i)
			switch specDec := decs.(type) {
			case *ast.GenDecl:
				if !serviceInterfaceFound {
					for j, spec := range specDec.Specs {
						util.Logf("\t\t---------- Specs %v ---------------\n", j)
						if typeSpec, ok := spec.(*ast.TypeSpec); ok {
							if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
								util.Logf("\t\t\t%v in Decls %v\n", typeSpec.Name.String(), i)
								//fmt.Println(typeSpec.Pos())
								//fmt.Println(typeSpec.End())
								_ = interfaceType
								declareIndexToDelete = i
								serviceInterfaceFound = true
							}
						}
					}
				}
			case *ast.FuncDecl:
				funcName := specDec.Name.String()
				if funcName != "NewBasicService" {
					if v.protobufMethods[funcName] == false {
						util.Logf("Handler method %v does not exist in Service description, deleting...", funcName)
						funcsToDelete = append(funcsToDelete, i)
					} else {
						v.handlerMethods[funcName] = true
					}
				}
			}
		}
		if serviceInterfaceFound {
			file.Decls = append(file.Decls[:declareIndexToDelete], file.Decls[declareIndexToDelete+1:]...)
		}
		for _, index := range funcsToDelete {
			file.Decls = append(file.Decls[:index], file.Decls[index+1:]...)
		}
	}
	return nil
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
