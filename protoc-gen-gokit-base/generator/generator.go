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

	log "github.com/Sirupsen/logrus"
	_ "github.com/davecgh/go-spew/spew"
)

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stderr)
}

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
	servicePath := wd + "/server/service.go"
	clientPath := wd + "/client/client_handler.go"
	for _, templateFile := range g.templateFileNames() {
		var generatedFilePath string
		var generatedCode string

		// If service.go does not exist, generate all files
		// If template file is not service.go then generate the file
		// If service.go exists and the template file is service.go then skip
		if _, err := os.Stat(clientPath); err == nil && filepath.Base(templateFile) == "client_handler.go" {
			log.Info("client/client_handler.go exists")
			continue
		}
		if _, err := os.Stat(servicePath); os.IsNotExist(err) || filepath.Base(templateFile) != "service.go" {
			if filepath.Ext(templateFile) != ".go" {
				log.WithField("Template file", templateFile).Debug("Skipping rendering non-buildable partial template")
				continue
			}

			// Remove "template_files/" so that generated files do not include that directory
			generatedFilePath = strings.TrimPrefix(templateFile, "template_files/")

			// Get the bytes from the file we are working on
			bytesOfFile, _ := g.templateFile(templateFile)

			generatedCode, _ = g.generate(templateFile, bytesOfFile)
		} else {

			log.Info("server/service.go exists")
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
				log.WithError(err).Fatal("server/service.go could not be parsed by go/parser into AST")
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

			ast.Walk(walker, fileAst)

			serviceCode = bytes.NewBuffer(nil)

			err = printer.Fprint(serviceCode, fset, fileAst)
			if err != nil {
				panic(err)
			}

			log.WithField("Code", serviceCode.String()).Debug("Server service handlers before template")
			for _, meth := range g.templateExec.Service.Methods {
				methName := meth.GetName()
				if walker.handlerMethods[methName] == false {
					log.WithField("Method", methName).Info("Rendering template for method")
					templateOut := g.applyTemplate("template_files/partial_template/handler.method", meth)
					serviceCode.Write(templateOut)
				} else {
					log.WithField("Method", methName).Info("Handler method already exists")
				}
			}

			templateOut := g.applyTemplate("template_files/partial_template/service.interface", g.templateExec)
			serviceCode.Write(templateOut)

			formatted, err := format.Source(serviceCode.Bytes())

			if err != nil {
				log.WithError(err).Warn("Code formatting error, generated service will not build, outputting unformatted code")
				// Set formatted to code so at least we get something to examine
				formatted = serviceCode.Bytes()
			}

			generatedFilePath = "server/service.go"

			generatedCode = string(formatted)

		}
		curResponseFile := plugin.CodeGeneratorResponse_File{
			Name:    &generatedFilePath,
			Content: &generatedCode,
		}

		codeGenFiles = append(codeGenFiles, &curResponseFile)
	}

	return codeGenFiles, nil
}

func (g *generator) applyTemplate(templateFile string, executor interface{}) []byte {
	templateBytes, _ := g.templateFile(templateFile)
	templateString := string(templateBytes)

	codeTemplate := template.Must(template.New(templateFile).Parse(templateString))
	outputBuffer := bytes.NewBuffer(nil)

	err := codeTemplate.Execute(outputBuffer, executor)
	if err != nil {
		log.WithError(err).Fatal("Template Error")
	}

	return outputBuffer.Bytes()
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

func (g *generator) generate(templateName string, templateBytes []byte) (string, error) {

	templateString := string(templateBytes)

	codeTemplate := template.Must(template.New(templateName).Parse(templateString))

	w := bytes.NewBuffer(nil)
	err := codeTemplate.Execute(w, g.templateExec)
	if err != nil {
		log.WithError(err).Fatal("Template Error")
	}

	code := w.String()
	code = formatCode(code)

	return code, nil
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

		log.WithField(
			"File Name", file.Name,
		).Debug("AST File")

		for i, decs := range file.Decls {
			switch specDec := decs.(type) {
			case *ast.GenDecl:
				if !serviceInterfaceFound {
					for j, spec := range specDec.Specs {

						log.WithFields(log.Fields{
							"Type":       "GenDecl",
							"Decl Index": i,
							"Spec Index": j,
						}).Debug("AST Spec")

						if typeSpec, ok := spec.(*ast.TypeSpec); ok {
							if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {

								log.WithFields(log.Fields{
									"Decls": typeSpec.Name.String(),
									"Index": i,
								}).Debug("Service interface found")

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
						log.WithField("Method", funcName).Info("Handler does not exist in proto service description, deleting...")
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
