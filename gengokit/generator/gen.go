package generator

import (
	"bytes"
	"go/format"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"

	"github.com/TuneLab/go-truss/gengokit"
	"github.com/TuneLab/go-truss/gengokit/astmodifier"
	templateFileAssets "github.com/TuneLab/go-truss/gengokit/template"

	"github.com/TuneLab/go-truss/svcdef"
	"github.com/TuneLab/go-truss/truss"
)

func init() {
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stderr)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})
}

// GenerateGokit returns a gokit service generated from a service definition (svcdef),
// the package to the root of the generated service goPackage, the package
// to the .pb.go service struct files (goPBPackage) and any prevously generated files.
func GenerateGokit(sd *svcdef.Svcdef, conf gengokit.Config) ([]truss.NamedReadWriter, error) {
	te, err := gengokit.NewTemplateExecutor(sd, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not create template executor")
	}

	fpm := make(map[string]io.Reader, len(conf.PreviousFiles))
	for _, f := range conf.PreviousFiles {
		fpm[f.Name()] = f
	}

	var codeGenFiles []truss.NamedReadWriter

	for _, templFP := range templateFileAssets.AssetNames() {
		file, err := generateResponseFile(templFP, te, fpm)
		if err != nil {
			return nil, errors.Wrap(err, "could not render template")
		}
		if file == nil {
			continue
		}

		codeGenFiles = append(codeGenFiles, file)
	}

	return codeGenFiles, nil
}

// generateResponseFile contains logic to choose how to render a template file
// based on path and if that file was generated previously. It accepts a
// template path to render, a templateExecutor to apply to the template, and a
// map of paths to files for the previous generation. It returns a
// truss.NamedReadWriter representing the generated file
func generateResponseFile(templFP string, te *gengokit.TemplateExecutor, prevGenMap map[string]io.Reader) (truss.NamedReadWriter, error) {
	var genCode io.Reader
	var err error

	// Get the actual path to the file rather than the template file path
	actualFP := templatePathToActual(templFP, te.PackageName)

	// Map of template paths to template rendering functions
	renderTempl := map[string]string{
		"NAME-service/handlers/server/server_handler.gotemplate": serverMethods,
	}

	// If we are rendering a template with a renderFunc and the file existed
	// previously then use that function
	if templ := renderTempl[templFP]; templ != "" {
		file := prevGenMap[actualFP]
		if file != nil {
			genCode, err = updateMethods(file, []byte(templ), te)
		}
	}

	if err != nil {
		return nil, errors.Wrap(err, "could not render template")
	}

	// if no code has been generated just apply the template
	if genCode == nil {
		genCode, err = applyTemplateFromPath(templFP, te)
	}

	if err != nil {
		return nil, errors.Wrap(err, "could not render template")
	}

	codeBytes, err := ioutil.ReadAll(genCode)
	if err != nil {
		return nil, err
	}

	// ignore error as we want to write the code either way to inspect after
	// writing to disk
	formattedCode := formatCode(codeBytes)

	var resp truss.SimpleFile

	// Set the path to the file and write the code to the file
	resp.Path = actualFP
	_, err = resp.Write(formattedCode)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// templatePathToActual accepts a templateFilePath and the packageName of the
// service and returns what the relative file path of what should be written to
// disk
func templatePathToActual(templFilePath, packageName string) string {
	// Switch "NAME" in path with packageName.
	// i.e. for packageName = addsvc; /NAME-service/NAME-server -> /addsvc-service/addsvc-server
	actual := strings.Replace(templFilePath, "NAME", packageName, -1)

	actual = strings.TrimSuffix(actual, "template")

	return actual
}

func updateMethods(handler io.Reader, templ []byte, te *gengokit.TemplateExecutor) (io.Reader, error) {
	svcFuncs := serviceFunctionsNames(te.Service.Methods)

	astMod := astmodifier.New(handler)

	astMod.RemoveFunctionsExecpt(svcFuncs)

	// Index handler functions, apply handler template for all function in
	// service definition that are not defined in handler
	currentFuncs := astMod.IndexFunctions()
	code := astMod.Buffer()

	trimmedTe := te.TrimServiceFuncs(currentFuncs)

	newFuncs, err := applyTemplate(templ, "", trimmedTe)
	if err != nil {
		return nil, errors.Wrap(err, "unable to apply methods template")
	}

	code.ReadFrom(newFuncs)

	return code, nil
}

// serviceFunctionNames returns a slice of function names which are in the
// definition files plus the function "NewService". Used for inserting and
// removing functions from previously generated handler files
func serviceFunctionsNames(methods []*svcdef.ServiceMethod) []string {
	var svcFuncs []string
	for _, m := range methods {
		svcFuncs = append(svcFuncs, m.Name)
	}
	svcFuncs = append(svcFuncs, "NewService")

	return svcFuncs

}

// applyTemplateFromPath calls applyTemplate with the template at templFilePath
func applyTemplateFromPath(templFilePath string, executor *gengokit.TemplateExecutor) (io.Reader, error) {
	templBytes, err := templateFileAssets.Asset(templFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to find template file: %v", templFilePath)
	}

	return applyTemplate(templBytes, templFilePath, executor)
}

func applyTemplate(templBytes []byte, templName string, executor *gengokit.TemplateExecutor) (io.Reader, error) {
	templateString := string(templBytes)

	codeTemplate, err := template.New(templName).Funcs(executor.FuncMap).Parse(templateString)
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
		// return code so at least we get something to examine
		return code
	}

	return formatted
}

var serverMethods string = `
{{ with $te := .}}
		{{range $i := $te.Service.Methods}}
		// {{.GetName}} implements Service.
		func (s {{$te.PackageName}}Service) {{.GetName}}(ctx context.Context, in *pb.{{GoName .RequestType.GetName}}) (*pb.{{GoName .ResponseType.GetName}}, error){
			var resp pb.{{GoName .ResponseType.GetName}}
			resp = pb.{{GoName .ResponseType.GetName}}{
				{{range $j := $i.ResponseType.Message.Fields -}}
					// {{GoName $j.GetName}}: 
				{{end -}}
			}
			return &resp, nil
		}
		{{end}}
{{- end}}
`
