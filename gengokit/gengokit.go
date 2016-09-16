package gengokit

import (
	"bytes"
	"go/format"
	"io"
	//"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	log "github.com/Sirupsen/logrus"
	generatego "github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/pkg/errors"

	"github.com/TuneLab/go-truss/gengokit/astmodifier"
	"github.com/TuneLab/go-truss/gengokit/clientarggen"
	"github.com/TuneLab/go-truss/gengokit/httptransport"
	templateFileAssets "github.com/TuneLab/go-truss/gengokit/template"

	"github.com/TuneLab/go-truss/deftree"
	"github.com/TuneLab/go-truss/truss/truss"
)

func init() {
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stderr)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})
}

type generator struct {
	templateExec templateExecutor
}

// templateExecutor is passed to templates as the executing struct its fields
// and methods are used to modify the template
type templateExecutor struct {
	// import path for the directory containing the definition .proto files
	ImportPath string
	// GRPC/Protobuff service, with all parameters and return values accessible
	Service    *deftree.ProtoService
	ClientArgs *clientarggen.ClientServiceArgs
	// A helper struct for generating http transport functionality.
	HTTPHelper *httptransport.Helper
}

// GenerateGokit accepts a deftree representing the ast of a group of .proto
// files, a []truss.NamedReadWriter representing files generated previously, and
// a goImportPath for templating go code imports
// GenerateGoCode returns the a []truss.NamedReadWriter representing a generated
// gokit microservice file structure
func GenerateGokit(dt deftree.Deftree, previousFiles []truss.NamedReadWriter, goImportPath string) ([]truss.NamedReadWriter, error) {
	service, err := getProtoService(dt)
	if err != nil {
		return nil, errors.Wrap(err, "no service found aborting generating gokit microservice")
	}

	importPath := goImportPath + "/" + dt.GetName() + "-service"
	g := newGenerator(service, importPath)
	files, err := g.GenerateResponseFiles(previousFiles, dt.GetName())

	if err != nil {
		return nil, errors.Wrap(err, "could not generate gokit microservice")
	}

	return files, nil
}

// getProtoService finds returns the service within a deftree.Deftree
func getProtoService(dt deftree.Deftree) (*deftree.ProtoService, error) {
	md := dt.(*deftree.MicroserviceDefinition)
	files := md.Files
	var service *deftree.ProtoService

	for _, file := range files {
		if len(file.Services) > 0 {
			service = file.Services[0]
		}
	}

	if service == nil {
		return nil, errors.New("no service found")
	}

	return service, nil
}

// newGenerator returns a new generator which generates a gokit microservice
func newGenerator(service *deftree.ProtoService, importPath string) *generator {
	return &generator{
		templateExec: templateExecutor{
			ImportPath: importPath,
			Service:    service,
			ClientArgs: clientarggen.New(service),
			HTTPHelper: httptransport.NewHelper(service),
		},
	}
}

// getFileContentByName searches though a []truss.NamedReadWriter and returns the
// contents of the file with the name n
func getFileByName(n string, files []truss.NamedReadWriter) io.Reader {
	for _, f := range files {
		if f.Name() == n {
			return f
		}
	}
	return nil
}

// updateServiceMethods will update the functions within an existing service.go
// file so it contains all the functions within svcFuncs
func (g *generator) updateServiceMethods(svcHandler io.Reader, svcFuncs []string) (outCode *bytes.Buffer, err error) {
	const svcMethodsTemplPath = "NAME-service/partial_template/service.methods"
	const svcInterfaceTemplPath = "NAME-service/partial_template/service.interface"

	astMod := astmodifier.New(svcHandler)

	//TODO: Discuss if functions should be removed from the service file, when using truss I did not like that it removed function I wrote myself
	//astMod.RemoveFunctionsExecpt(svcFuncs)
	astMod.RemoveInterface("Service")

	log.WithField("Code", astMod.String()).Debug("Server service handlers before template")

	// Index the handler functions, apply handler template for all function in service definition that are not defined in handler
	currentFuncs := astMod.IndexFunctions()
	code := astMod.Buffer()

	err = g.applyTemplateForMissingMeths(svcMethodsTemplPath, currentFuncs, code)
	if err != nil {
		return nil, errors.Wrap(err, "unable to apply service methods template")
	}

	// Insert updated Service interface
	outBuf, err := applyTemplateFromPath(svcInterfaceTemplPath, g.templateExec)
	if err != nil {
		return nil, errors.Wrap(err, "unable to apply service interface template")
	}

	code.Write(outBuf.Bytes())

	return code, nil
}

// updateClientMethods will update the functions within an existing
// client_handler.go file so that it contains exactly the fucntions passed in
// svcFuncs, no more, no less.
func (g *generator) updateClientMethods(clientHandler io.Reader, svcFuncs []string) (outCode *bytes.Buffer, err error) {
	const clientMethodsTemplPath = "NAME-service/partial_template/client_handler.methods"
	astMod := astmodifier.New(clientHandler)

	// Remove functions no longer in definition
	astMod.RemoveFunctionsExecpt(svcFuncs)

	log.WithField("Code", astMod.String()).Debug("Client handlers before template")

	// Index handler functions, apply handler template for all function in
	// service definition that are not defined in handler
	currentFuncs := astMod.IndexFunctions()
	code := astMod.Buffer()

	err = g.applyTemplateForMissingMeths(clientMethodsTemplPath, currentFuncs, code)

	if err != nil {
		return nil, errors.Wrap(err, "unable to apply client methods template")
	}

	return code, nil
}

// GenerateResponseFiles applies all template files for the generated
// microservice and returns a slice containing each templated file as a
// truss.NamedReadWriter.
func (g *generator) GenerateResponseFiles(previousFiles []truss.NamedReadWriter, packageName string) ([]truss.NamedReadWriter, error) {
	// Paths to handler files that may be modified programmatically
	serviceHandlerFilePath := packageName + "/handlers/server/server_handler.go"
	clientHandlerFilePath := packageName + "/handlers/client/client_handler.go"

	var codeGenFiles []truss.NamedReadWriter

	// serviceFunctions is used later as the master list of all service methods
	// which should exist within the `server/service.go` and
	// `client/client_handler.go` files.
	var serviceFunctions []string
	for _, meth := range g.templateExec.Service.Methods {
		serviceFunctions = append(serviceFunctions, meth.GetName())
	}
	serviceFunctions = append(serviceFunctions, "NewBasicService")

	serviceHandlerFile := getFileByName(serviceHandlerFilePath, previousFiles)
	clientHandlerFile := getFileByName(clientHandlerFilePath, previousFiles)

	for _, templateFilePath := range templateFileAssets.AssetNames() {
		if filepath.Ext(templateFilePath) != ".gotemplate" {
			log.WithField("Template file", templateFilePath).Debug("Skipping rendering non-buildable partial template")
			continue
		}

		var generatedFilePath string
		var generatedCode *bytes.Buffer
		var err error

		if templateFilePath == serviceHandlerFilePath+"template" && serviceHandlerFile != nil {
			// If there's an existing service file, update its contents

			generatedFilePath = serviceHandlerFilePath
			generatedCode, err = g.updateServiceMethods(serviceHandlerFile, serviceFunctions)

			if err != nil {
				return nil, errors.Wrap(err, "could not modifiy service handler file")
			}

		} else if templateFilePath == clientHandlerFilePath+"template" && clientHandlerFile != nil {
			// If there's an existing client_handler file, update its contents

			generatedFilePath = clientHandlerFilePath
			generatedCode, err = g.updateClientMethods(clientHandlerFile, serviceFunctions)

			if err != nil {
				return nil, errors.Wrap(err, "could not modifiy client handler file")
			}

		} else {
			generatedFilePath = templateFilePath
			// Change file path from .gotemplate to .go
			generatedFilePath = strings.TrimSuffix(generatedFilePath, "template")

			generatedCode, err = applyTemplateFromPath(templateFilePath, g.templateExec)
			if err != nil {
				return nil, errors.Wrap(err, "could not render template")
			}
		}

		// Turn code buffer into string and format it
		formattedCode := formatCode(generatedCode.Bytes())

		var resp truss.SimpleFile
		// Switch "NAME" in path with packageName.
		//i.e. for packageName = addsvc; /NAME-service/NAME-server -> /addsvc-service/addsvc-server
		resp.Path = strings.Replace(generatedFilePath, "NAME", packageName, -1)

		resp.Write(formattedCode)

		codeGenFiles = append(codeGenFiles, &resp)
	}

	return codeGenFiles, nil
}

// applyTemplateForMissingMeths accepts a funcIndex which represents functions
// already present in a gofile, applyTemplateForMissingMeths compares this map
// to the functions defined in the service and renders the template with the
// path of templPath, and appends this to passed code
func (g *generator) applyTemplateForMissingMeths(templPath string, funcIndex map[string]bool, code *bytes.Buffer) error {
	var methodsToTemplate []*deftree.ServiceMethod
	for _, meth := range g.templateExec.Service.Methods {
		methName := meth.GetName()
		if funcIndex[methName] == false {
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
	templateOut, err := applyTemplateFromPath(templPath, templateExecWithOnlyMissingMethods)
	if err != nil {
		return errors.Wrapf(err, "could not apply template for missing methods: %v", templPath)
	}

	_, err = code.Write(templateOut.Bytes())
	if err != nil {
		return errors.Wrap(err, "could not append rendered template to code")
	}

	return nil
}

// applyTemplateFromPath accepts a path to a template and an interface to execute on that template
// returns a *bytes.Buffer containing the results of that execution
func applyTemplateFromPath(templFilePath string, executor interface{}) (*bytes.Buffer, error) {

	templBytes, err := templateFileAssets.Asset(templFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to find template file: %v", templFilePath)
	}

	return applyTemplate(templBytes, templFilePath, executor)
}

// applyTemplate accepts a []byte, and a string representing the content and
// name of a template and an interface to execute on that template returns a
// *bytes.Buffer containing the results of that execution
func applyTemplate(templBytes []byte, templName string, executor interface{}) (*bytes.Buffer, error) {

	templateString := string(templBytes)

	funcMap := template.FuncMap{
		"ToLower":    strings.ToLower,
		"Title":      strings.Title,
		"GoName":     generatego.CamelCase,
		"TrimPrefix": strings.TrimPrefix,
	}

	codeTemplate, err := template.New(templName).Funcs(funcMap).Parse(templateString)

	if err != nil {
		return nil, errors.Wrap(err, "could not create template")
	}

	outputBuffer := bytes.NewBuffer(nil)
	err = codeTemplate.Execute(outputBuffer, executor)
	if err != nil {
		return nil, errors.Wrap(err, "template error")
	}

	return outputBuffer, nil
}

// formatCode takes a string representing golang code and attempts to return a
// formated copy of that code.  If formatting fails, a warning is logged and
// the original code is returned.
func formatCode(code []byte) []byte {
	formatted, err := format.Source(code)

	if err != nil {
		log.WithError(err).Warn("Code formatting error, generated service will not build, outputting unformatted code")
		// Set formatted to code so at least we get something to examine
		formatted = code
	}

	return formatted
}
